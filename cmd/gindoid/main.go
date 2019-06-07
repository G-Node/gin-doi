package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	"github.com/G-Node/libgin/libgin"
	docopt "github.com/docopt/docopt-go"
	log "github.com/sirupsen/logrus"
)

const usage = `gindoid: DOI service for preparing GIN repositories for publication
Usage:
  gindoid [--debug]

  --debug              Print debug messages
`

// TODO: Make non-global
var doibase string

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

	configu, err := loadconfig()
	if err != nil {
		log.Errorf("Startup failed: %v", err)
		os.Exit(-1)
	}
	j, _ := json.MarshalIndent(configu, "", "  ")
	fmt.Println(string(j))

	jobQueue := make(chan DOIJob, configu.MaxQueue)
	dispatcher := NewDispatcher(jobQueue, configu.MaxWorkers)
	dispatcher.Run(NewWorker)

	// Start the HTTP handlers.
	http.Handle("/", http.RedirectHandler(configu.Storage.StoreURL, http.StatusMovedPermanently))
	http.HandleFunc("/register", func(w http.ResponseWriter, r *http.Request) {
		log.Debugf("Got request: %s", r.URL.String())
		InitDOIJob(w, r, nil, nil, configu.TemplatePath, nil, configu.Key, configu)
	})
	http.HandleFunc("/do/", func(w http.ResponseWriter, r *http.Request) {
		DoDOIJob(w, r, jobQueue, LocalStorage{}, nil, configu)
	})
	http.Handle("/assets/",
		http.StripPrefix("/assets/", http.FileServer(http.Dir("/assets"))))

	port := libgin.ReadConfDefault("port", "10443")
	fmt.Printf("Listening for connections on port %s\n", port)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%s", port), nil))
}
