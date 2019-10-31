package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	docopt "github.com/docopt/docopt-go"
	log "github.com/sirupsen/logrus"
)

const usage = `gindoid: DOI service for preparing GIN repositories for publication
Usage:
  gindoid [--debug]

  --debug              Print debug messages
`

func main() {
	args, err := docopt.Parse(usage, nil, true, "gin doi 0.1a", false)
	if err != nil {
		log.Printf("Error while parsing command line: %s", err.Error())
		os.Exit(-1)
	}
	//Debugging?
	debug := args["--debug"].(bool)
	if debug {
		log.SetLevel(log.DebugLevel)
		log.SetFormatter(&log.TextFormatter{ForceColors: true})
	}

	log.Debug("Starting up")

	config, err := loadconfig()
	if err != nil {
		log.Errorf("Startup failed: %v", err)
		os.Exit(-1)
	}

	if debug {
		// Pretty print configuration when debugging, but hide sensitive stuff
		cc := *config
		cc.Key = "[HIDDEN]"
		cc.GIN.Password = "[HIDDEN]"
		j, _ := json.MarshalIndent(cc, "", "  ")
		log.Debug(string(j))
	}

	log.Debugf("Logging in to GIN (%s) as %s", config.GIN.Session.WebAddress(), config.GIN.Username)
	err = config.GIN.Session.Login(config.GIN.Username, config.GIN.Password, "gin-doi")
	if err != nil {
		log.Error(err)
		os.Exit(-1)
	}

	defer config.GIN.Session.Logout()

	jobQueue := make(chan DOIJob, config.MaxQueue)
	dispatcher := NewDispatcher(jobQueue, config.MaxWorkers)
	dispatcher.Run(NewWorker)

	// Start the HTTP handlers.

	// Root redirects to storage URL (DOI listing page)
	http.Handle("/", http.RedirectHandler(config.Storage.StoreURL, http.StatusMovedPermanently))

	// register renders the info page with the registration button
	http.HandleFunc("/register", func(w http.ResponseWriter, r *http.Request) {
		log.Debugf("Got request: %s", r.URL.String())
		InitDOIJob(w, r, config)
	})

	// do starts the registration job
	http.HandleFunc("/do/", func(w http.ResponseWriter, r *http.Request) {
		DoDOIJob(w, r, jobQueue, config)
	})

	// assets fetches static assets using a custom FileSystem
	assetserver := http.FileServer(NewAssetFS("/assets"))
	http.Handle("/assets/", http.StripPrefix("/assets/", assetserver))

	fmt.Printf("Listening for connections on port %d\n", config.Port)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", config.Port), nil))
}
