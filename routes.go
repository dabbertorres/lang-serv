package main

import (
	"archive/tar"
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"mime"
	"net/http"
	"path"
	"strings"

	"github.com/docker/docker/api/types"
	"github.com/gorilla/mux"
)

func Route() (router *mux.Router) {
	router = mux.NewRouter()

	router.Path("/").
		Methods(http.MethodGet).
		HandlerFunc(homeHandler)

	router.Path("/style.css").
		Methods(http.MethodGet).
		HandlerFunc(staticFileHandler("app/style.css"))

	router.Path("/lang.js").
		Methods(http.MethodGet).
		HandlerFunc(staticFileHandler("app/lang.js"))

	router.Path("/new.svg").
		Methods(http.MethodGet).
		HandlerFunc(staticFileHandler("app/new.svg"))

	router.Path("/delete.svg").
		Methods(http.MethodGet).
		HandlerFunc(staticFileHandler("app/delete.svg"))

	languageRouter := router.Path("/{language}/{version}").Subrouter()

	// opening a new session for a language
	languageRouter.
		Methods(http.MethodGet).
		HandlerFunc(languageGetHandler)

	// running/building with a language
	languageRouter.
		Methods(http.MethodPost).
		Headers("Content-Type", "application/json").
		HandlerFunc(languagePostHandler)

	// TODO sending all files/etc every POST is a little overkill

	// redirect to /{language}/latest
	router.Path("/{language}").
		Methods(http.MethodGet).
		HandlerFunc(languageLatestSymlinkHandler)

	// TODO method for sharing a session
	// a container commit (create an image from a container) would be perfect.

	return
}

func homeHandler(w http.ResponseWriter, r *http.Request) {
	err := IndexPage(w, IndexPageData{})
	if err != nil {
		log.Printf("%s: %s\n", r.RequestURI, err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func staticFileHandler(filename string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		buf, err := ioutil.ReadFile(filename)
		if err != nil {
			log.Printf("%s: %s\n", r.RequestURI, err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", mime.TypeByExtension(path.Ext(filename)))
		w.Write(buf)
	}
}

func languageGetHandler(w http.ResponseWriter, r *http.Request) {
	session, err := sessionStore.Get(r, sessionCookieKey)
	if err != nil {
		log.Printf("%s: %s\n", r.RequestURI, err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if session.IsNew {
		err = NewSession(session, r)
		if err != nil {
			log.Printf("%s: %s\n", r.RequestURI, err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		err = session.Save(r, w)
		if err != nil {
			log.Printf("%s: %s\n", r.RequestURI, err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	err = LanguagePage(w, LanguagePageData{
		Language: mux.Vars(r)["language"],
		Version:  mux.Vars(r)["version"],
	})
	if err != nil {
		log.Printf("%s: %s\n", r.RequestURI, err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func languagePostHandler(w http.ResponseWriter, r *http.Request) {
	dec := json.NewDecoder(r.Body)

	session, err := sessionStore.Get(r, sessionCookieKey)
	if err != nil {
		log.Printf("%s: %s\n", r.RequestURI, err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if session.IsNew {
		log.Printf("%s: %s\n", r.RequestURI, "new session in POST")
		session.Options.MaxAge = -1
		http.Error(w, err.Error(), http.StatusForbidden)
		return
	}

	var (
		post         = LanguagePost{}
		ctnr         = session.Values[sessionContainerKey].(string)
		filesArchive = bytes.NewBuffer(nil)
		tarW         = tar.NewWriter(filesArchive)
	)

	if err := dec.Decode(&post); err != nil {
		log.Printf("[ERROR] %s: %s\n", r.RequestURI, err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	for _, f := range post.Files {
		err = tarW.WriteHeader(&tar.Header{
			Name: f.Name,
			Mode: 0644,
			Size: int64(len(f.Data)),
		})
		if err != nil {
			log.Printf("[ERROR] %s: writing tar file header: %s\n", r.RequestURI, err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		_, err = tarW.Write([]byte(f.Data))
		if err != nil {
			log.Printf("[ERROR] %s: writing tar file: %s\n", r.RequestURI, err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
	err = tarW.Close()
	if err != nil {
		log.Printf("[ERROR] %s: closing tar file: %s\n", r.RequestURI, err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	err = docker.CopyToContainer(r.Context(), ctnr, containerWorkingDirectory,
		filesArchive, types.CopyToContainerOptions{})
	if err != nil {
		log.Printf("[ERROR] %s: copying files to container: %s\n", r.RequestURI, err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	execCfg := types.ExecConfig{
		AttachStderr: true,
		AttachStdout: true,
		Cmd:          strings.Fields(strings.TrimSpace(post.Cmd)),
	}

	execResp, err := docker.ContainerExecCreate(r.Context(), ctnr, execCfg)
	if err != nil {
		log.Printf("[ERROR] docker.ContainerExecCreate(): %s: %s. Command: %v\n", r.RequestURI, err, execCfg.Cmd)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	hijack, err := docker.ContainerExecAttach(r.Context(), execResp.ID, execCfg)
	if err != nil {
		log.Printf("[ERROR] docker.ContainerExecAttach(): %s: %s\n", r.RequestURI, err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	output, err := ioutil.ReadAll(hijack.Reader)
	if err != nil {
		log.Printf("[ERROR] Reading container output: %s: %s\n", r.RequestURI, err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// don't write the first 8 bytes because they seem to be an 8 byte integer representing the string's length
	fmt.Fprint(w, string(output[8:]))
}

func languageLatestSymlinkHandler(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, fmt.Sprintf("/%s/latest", mux.Vars(r)["language"]), http.StatusFound)
}
