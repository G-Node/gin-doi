package main

import (
	"fmt"
	"net/http"
	"os"
	"strconv"

	log "github.com/Sirupsen/logrus"
	"github.com/docopt/docopt-go"
)

const usage = `gindoid: DOI service for preparing GIN repositories for publication
Usage:
  gindoid [--maxworkers=<n> --maxqueue=<n> --port=<port> --source=<url> --gitsource=<url>
           --oauthserver=<url> --target=<dir> --storeurl=<url> --mailserver=<host:port> --mailfrom=<address>
           --mailto=<address> --doibase=<prefix> --sendmail --debug --templates=<path> --xmlurl=<url>] --key=<key>

Options:
  --maxworkers=<n>                 The number of workers to start [default: 3]
  --maxqueue=<n>                   The size of the job queue [default: 100]
  --port=<port>                    The server port [default: 8083]
  --source=<url>                   The server address from which data can be read [default: https://web.gin.g-node.org]
  --gitsource=<url>                The git server address from which data can be cloned [default: ssh://git@gin.g-node.org]
  --oauthserver=<url>              The server of the repo service [default: https://web.gin.g-node.org]
  --target=<dir>                   The location for long term storage [default: data]
  --storeurl=<url>                 The base URL for storage [default: http://doid.gin.g-node.org/]
  --mailserver=<host:port>         The mail server address (:and port) [default: localhost:25]
  --mailfrom=<address>             The mail from address [default: no-reply@g-node.org]
  --mailto=<address>               The mail address to send info to [default: dev@g-node.org]
  --doibase=<prefix>               The DOI prefix [default: 10.12751/g-node.]
  --sendmail                       Whether mail notifications should really be sent (otherwise just print them)
  --debug                          Whether debug messages shall be printed
  --templates=<path>               Path to the templates [default: tmpl]
  --xmlurl=<url>                   URI of the datacite XML [default: gin.g-node.org:/data/doid]
  --key=<key>                      Key used to decrypt token
`

func main() {
	args, err := docopt.Parse(usage, nil, true, "gin doi 0.1a", false)
	if err != nil {
		log.Printf("Error while parsing command line: %+v", err)
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
	ginurl := args["--source"].(string)
	giturl := args["--gitsource"].(string)
	log.Debugf("gin: %s -- git: %s", ginurl, giturl)
	ds := &GogsDataSource{GinURL: ginurl, GinGitURL: giturl}

	// doi provider
	doibase := args["--doibase"].(string)
	log.Debugf("doibase: %s", doibase)
	dp := GnodeDOIProvider{APIURI: "", DOIBase: doibase}

	//Setup storage
	mailserver := args["--mailserver"].(string)
	mailfrom := args["--mailfrom"].(string)
	mailto := args["--mailto"].(string)
	sendmail := args["--sendmail"].(bool)
	mServer := MailServer{
		Address:   mailserver,
		From:      mailfrom,
		DoSend:    sendmail,
		Recipient: mailto,
	}
	log.Debugf("Mail configuration: %+v", mServer)

	target := args["--target"].(string)
	storeurl := args["--storeurl"].(string)
	templates := args["--templates"].(string)
	xmlurl := args["--xmlurl"].(string)
	storage := LocalStorage{
		Path:   target,
		Source: ds, HTTPBase: storeurl,
		DProvider:    dp,
		MServer:      &mServer,
		TemplatePath: templates,
		SCPURL:       xmlurl,
	}
	log.Debugf("LocalStorage configuration: %+v", storage)

	// setup authentication
	oAuthAddress := args["--oauthserver"].(string)
	op := GogsOAuthProvider{
		URI:      fmt.Sprintf("%s/api/v1/user", oAuthAddress),
		TokenURL: "",
		KeyURL:   fmt.Sprintf("%s/api/v1/user/keys", oAuthAddress),
	}
	log.Debugf("OAuth configuration: %+v", op)

	key := args["--key"].(string)

	// Create the job queue.
	maxQ, err := strconv.Atoi(args["--maxqueue"].(string))
	if err != nil {
		log.Printf("Error while parsing maxqueue flag: %s", err.Error())
		log.Print("Using default")
		maxQ = 100
	}
	jobQueue := make(chan DOIJob, maxQ)
	// Start the dispatcher.
	maxW, err := strconv.Atoi(args["--maxworkers"].(string))
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
		InitDOIJob(w, r, ds, &op, storage.TemplatePath, &storage, key)
	})
	http.HandleFunc("/do/", func(w http.ResponseWriter, r *http.Request) {
		DoDOIJob(w, r, jobQueue, storage, &op)
	})
	http.Handle("/assets/",
		http.StripPrefix("/assets/", http.FileServer(http.Dir("/assets"))))

	port := args["--port"].(string)
	fmt.Printf("Listening for connections on port %s\n", port)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%s", port), nil))
}
