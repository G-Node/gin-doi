package main

import (
	"fmt"
	"net/http"
	"os"
	"strconv"

	"github.com/G-Node/libgin"
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

	// Setup data source
	ginurl := libgin.ReadConf("ginurl")
	giturl := libgin.ReadConf("giturl")
	log.Debugf("gin: %s -- git: %s", ginurl, giturl)
	ds := DataSource{GinURL: ginurl, GinGitURL: giturl}

	doibase = libgin.ReadConf("doibase")
	log.Debugf("doibase: %s", doibase)

	// Setup storage
	mailserver := libgin.ReadConf("mailserver")
	mailfrom := libgin.ReadConf("mailfrom")
	sendmail := true // TODO: Remove option
	mailtofile := libgin.ReadConf("mailtofile")
	mServer := MailServer{
		Address:   mailserver,
		From:      mailfrom,
		DoSend:    sendmail,
		EmailList: mailtofile,
	}
	log.Debugf("Mail configuration: %+v", mServer)

	target := libgin.ReadConf("target")
	storeurl := libgin.ReadConf("storeurl")
	templates := libgin.ReadConf("templates")
	xmlurl := libgin.ReadConf("xmlurl")
	knownhosts := libgin.ReadConf("knownhosts")
	storage := LocalStorage{
		Path:         target,
		Source:       ds,
		HTTPBase:     storeurl,
		MServer:      &mServer,
		TemplatePath: templates,
		SCPURL:       xmlurl,
		KnownHosts:   knownhosts,
	}
	log.Debugf("LocalStorage configuration: %+v", storage)

	// setup authentication
	oAuthAddress := libgin.ReadConf("oauthserver")
	op := OAuthProvider{
		URI:      fmt.Sprintf("%s/api/v1/user", oAuthAddress),
		TokenURL: "",
		KeyURL:   fmt.Sprintf("%s/api/v1/user/keys", oAuthAddress),
	}
	log.Debugf("OAuth configuration: %+v", op)

	key := libgin.ReadConf("key")

	// Create the job queue.
	maxQ, err := strconv.Atoi(libgin.ReadConfDefault("maxqueue", "100"))
	if err != nil {
		log.Printf("Error while parsing maxqueue flag: %s", err.Error())
		log.Print("Using default")
		maxQ = 100
	}
	jobQueue := make(chan DOIJob, maxQ)
	// Start the dispatcher.
	maxW, err := strconv.Atoi(libgin.ReadConfDefault("maxworkers", "3"))
	if err != nil {
		log.Printf("Error while parsing maxworkers flag: %s", err.Error())
		log.Print("Using default")
		maxW = 3
	}

	log.Debugf("Max queue: %d   Max workers: %d", maxQ, maxW)

	dispatcher := NewDispatcher(jobQueue, maxW)
	dispatcher.Run(NewWorker)

	// Start the HTTP handlers.
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		log.Debugf("Got request: %s", r.URL.String())
		InitDOIJob(w, r, &ds, &op, storage.TemplatePath, &storage, key)
	})
	http.HandleFunc("/do/", func(w http.ResponseWriter, r *http.Request) {
		DoDOIJob(w, r, jobQueue, storage, &op)
	})
	http.Handle("/assets/",
		http.StripPrefix("/assets/", http.FileServer(http.Dir("/assets"))))

	port := libgin.ReadConfDefault("port", "10444")
	fmt.Printf("Listening for connections on port %s\n", port)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%s", port), nil))
}
