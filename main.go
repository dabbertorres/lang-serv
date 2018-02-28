package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/docker/docker/client"
	"github.com/gorilla/handlers"
	"github.com/gorilla/sessions"
)

var (
	docker       *client.Client
	sessionStore *sessions.CookieStore

	exitCode = 0

	logToFile bool
	authKeyFile = "auth.keys"
)

func init() {
	flag.BoolVar(&logToFile, "log-file", false, "passing enables duplicating logs to a log file")
	flag.StringVar(&authKeyFile, "auth-file", authKeyFile, "specify a file to use for storing session authentication keys")

	var err error
	docker, err = client.NewEnvClient()
	if err != nil {
		log.Println("[FATAL] Creating docker client connection error:", err)
		os.Exit(1)
	}
}

func main() {
	defer os.Exit(exitCode)
	defer docker.Close()

	/* logging config */

	logOut := io.Writer(os.Stdout)
	if logToFile {
		logFile, err := os.Create(time.Now().Format("2006-01-02 15_04_05 -0700.log"))
		if err == nil {
			defer logFile.Close()
			logOut = io.MultiWriter(os.Stdout, logFile)
		} else {
			log.Println("[WARN] Unable to create log file:", err)
			log.Println("[INFO] Continuing without file logging.")
		}
	}
	log.SetOutput(logOut)

	/* sessions */

	authFile, err := LoadAuthFile(authKeyFile, false, 64)
	if err != nil {
		log.Fatalln("[FATAL] LoadAuthFile():", err)
	}

	sessionStore = sessions.NewCookieStore(authFile.AsKeyPairs()...)
	sessionStore.MaxAge(int(24 * time.Hour))

	/* app files */

	if err := LoadTemplates(); err != nil {
		log.Println("[ERROR] Loading app files:", err)
		exitCode = 1
		return
	}
	defer CloseTemplates()

	/* routing */

	router := Route()
	router.Use(func(next http.Handler) http.Handler { return handlers.CombinedLoggingHandler(logOut, next) })

	// requests to the server for a specific language create a session that launches a Docker container
	// configured for said language
	// container persists until session ends.
	// could be extended to allow for account creation and enable persistence

	server := &http.Server{
		Addr:         ":8080",
		IdleTimeout:  15 * time.Second,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		Handler:      router,
	}

	if err := server.ListenAndServe(); err != http.ErrServerClosed {
		fmt.Println("Unexpected ListenAndServe() error:", err)
		exitCode = 1
	}
}
