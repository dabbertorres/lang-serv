package main

import (
	"html/template"
	"io"
	"log"
	"sync"

	"github.com/fsnotify/fsnotify"
)

var (
	templateFiles = []string{
		"app/index.html",
		"app/lang.html",
	}

	watcher *fsnotify.Watcher

	htmlTemplates  *template.Template
	templatesMutex sync.RWMutex
)

func LoadTemplates() (err error) {
	htmlTemplates, err = template.ParseGlob("app/*.html")
	if err != nil {
		return
	}

	watcher, err = fsnotify.NewWatcher()
	if err == nil {
		for _, f := range templateFiles {
			err = watcher.Add(f)
			if err != nil {
				watcher.Close()
				return
			}
		}

		go watcherErrorLog()
		go watchFiles()
	} else {
		log.Println("[ERROR] File watch setup failed:", err)
		log.Println("[INFO] Continuing without file watching and auto-reloading.")
	}

	return
}

func CloseTemplates() {
	if watcher != nil {
		watcher.Close()
	}
}

func IndexPage(w io.Writer, data IndexGetData) error {
	templatesMutex.RLock()
	defer templatesMutex.RUnlock()
	return htmlTemplates.ExecuteTemplate(w, "index.html", data)
}

func LanguagePage(w io.Writer, data LanguageGetData) error {
	templatesMutex.RLock()
	defer templatesMutex.RUnlock()
	return htmlTemplates.ExecuteTemplate(w, "lang.html", data)
}

func watchFiles() {
	for range watcher.Events {
		var err error
		// lazy - just reload all of the template files...
		htmlTemplates, err = template.ParseGlob("app/*.html")
		if err != nil {
			log.Println("[ERROR] Loading modified templates:", err)
		}
	}
}

func watcherErrorLog() {
	for err := range watcher.Errors {
		log.Println("[ERROR] fsnotify:", err)
	}
}
