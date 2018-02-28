package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/gorilla/mux"
	"github.com/gorilla/sessions"
)

const (
	sessionCookieKey          = "compilerRPC-session-key"
	sessionContainerKey       = "compilerRPC-container-id"
	containerWorkingDirectory = "/work-area/"
)

func SessionRun(session *sessions.Session, r *http.Request) error {
	// if this session isn't new, then it (likely) already has a container running
	// let's stop that container
	if !session.IsNew && session.Values[sessionContainerKey] != "" {
		go func(key string) {
			err := docker.ContainerStop(context.Background(), key, nil)
			if err != nil {
				log.Printf("[WARN]: Error attempting to stop container %s: %s\n", key, err)
			}
		}(session.Values[sessionContainerKey].(string))
	}

	// we need to launch a new container for the matching image
	refStr := fmt.Sprintf("%s:%s", mux.Vars(r)["language"], mux.Vars(r)["version"])

	imgFilter := filters.NewArgs()
	imgFilter.Add("reference", refStr)

	images, err := docker.ImageList(r.Context(), types.ImageListOptions{
		All:     false,
		Filters: imgFilter,
	})
	if err != nil {
		return err
	}

	// if no results, we need to pull the image
	if len(images) == 0 {
		// TODO may want to run this outside of r's context, to avoid timing out (see next comment)
		// TODO run this in a goroutine, let user know the container is being prepared, and live update them when ready
		read, err := docker.ImagePull(r.Context(), refStr, types.ImagePullOptions{})
		if err != nil {
			return err
		}
		io.Copy(os.Stdout, read)
	}

	ctnr, err := docker.ContainerCreate(r.Context(), &container.Config{
		Image:        refStr,
		WorkingDir:   containerWorkingDirectory,
		AttachStderr: true,
		AttachStdin:  true,
		AttachStdout: true,
		OpenStdin:    true,
	}, nil, nil, "")
	if err != nil {
		log.Printf("[ERROR] ContainerCreate(%s): %s\n", refStr, err)
		return err
	}

	for _, w := range ctnr.Warnings {
		log.Printf("[WARN] ContainerCreate(%s): %s\n", refStr, w)
	}

	updateOk, err := docker.ContainerUpdate(r.Context(), ctnr.ID, container.UpdateConfig{
		Resources: container.Resources{
			CPUShares: 256,
			Memory: 64 * 1024 * 1024, // 64 MB
			NanoCPUs: int64(30 * time.Second),
			DiskQuota: 16 * 1024, // 16 KB
		},
	})
	if err != nil {
		log.Printf("[ERROR] ContainerUpdate(%s): %s\n", refStr, err)
		return err
	}

	for _, w := range updateOk.Warnings {
		log.Printf("[WARN] ContainerUpdate(%s): %s\n", refStr, w)
	}

	err = docker.ContainerStart(r.Context(), ctnr.ID, types.ContainerStartOptions{})
	if err != nil {
		docker.ContainerRemove(r.Context(), ctnr.ID, types.ContainerRemoveOptions{})
		log.Printf("[ERROR] ContainerStart(%s): %s\n", refStr, err)
		return err
	}

	session.Values[sessionContainerKey] = ctnr.ID

	return nil
}
