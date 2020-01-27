package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

	docopt "github.com/docopt/docopt-go"
)

const usage = `gindoid: DOI service for preparing GIN repositories for publication
Usage:
  gindoid

  No arguments are currently supported.
`

var (
	appversion string
	build      string
	commit     string
)

func init() {
	if appversion == "" {
		appversion = "[dev]"
	}
}

func main() {

	verstr := fmt.Sprintf("GIN DOI %s Build %s (%s)", appversion, build, commit)
	_, err := docopt.Parse(usage, nil, true, verstr, false)
	if err != nil {
		// NOTE: Keeping arg parsing around for upcoming CL functions
		log.Printf("Error while parsing command line: %s", err.Error())
		os.Exit(-1)
	}

	log.Printf("Starting up %s", verstr)

	config, err := loadconfig()
	if err != nil {
		log.Printf("Startup failed: %v", err)
		os.Exit(-1)
	}

	// Pretty print configuration for debugging, but hide sensitive stuff
	cc := *config
	cc.Key = "[HIDDEN]"
	cc.GIN.Password = "[HIDDEN]"
	j, _ := json.MarshalIndent(cc, "", "  ")
	log.Print(string(j))

	log.Printf("Logging in to GIN (%s) as %s", config.GIN.Session.WebAddress(), config.GIN.Username)
	err = config.GIN.Session.Login(config.GIN.Username, config.GIN.Password, "gin-doi")
	if err != nil {
		log.Print(err)
		os.Exit(-1)
	}

	defer config.GIN.Session.Logout()

	jobQueue := make(chan DOIJob, config.MaxQueue)
	dispatcher := newDispatcher(jobQueue, config.MaxWorkers)
	dispatcher.run(newWorker)

	// Start the HTTP handlers.

	// Root redirects to storage URL (DOI listing page)
	http.Handle("/", http.RedirectHandler(config.Storage.StoreURL, http.StatusMovedPermanently))

	// register renders the info page with the registration button
	http.HandleFunc("/register", func(w http.ResponseWriter, r *http.Request) {
		log.Printf("Got request: %s", r.URL.String())
		renderRequestPage(w, r, config)
	})

	// submit starts the registration job
	http.HandleFunc("/submit", func(w http.ResponseWriter, r *http.Request) {
		startDOIRegistration(w, r, jobQueue, config)
	})

	// assets fetches static assets using a custom FileSystem
	assetserver := http.FileServer(newAssetFS("/assets"))
	http.Handle("/assets/", http.StripPrefix("/assets/", assetserver))

	fmt.Printf("Listening for connections on port %d\n", config.Port)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", config.Port), nil))
}
