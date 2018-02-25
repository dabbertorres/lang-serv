package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"

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

func NewSession(session *sessions.Session, r *http.Request) error {
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

	for _, w := range ctnr.Warnings {
		log.Printf("[WARN]: ContainerCreate(%s): %s\n", refStr, w)
	}

	// switch contexts in order to keep the container running after this request finishes!
	err = docker.ContainerStart(r.Context(), ctnr.ID, types.ContainerStartOptions{})
	if err != nil {
		docker.ContainerRemove(r.Context(), ctnr.ID, types.ContainerRemoveOptions{})
		return err
	}

	session.Values[sessionContainerKey] = ctnr.ID

	return err
}
