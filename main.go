package main

import (
	"github.com/G-Node/gin-doi/src"
	log "github.com/Sirupsen/logrus"
	"github.com/docopt/docopt-go"
	"net/http"
	"os"
	"strconv"
	"io/ioutil"
	"encoding/json"
	"github.com/G-Node/gin-core/gin"
	"fmt"
)

func main() {
	usage := `gin doi.
Usage:
  gin-doi [--max_workers=<max_workers> --max_queue_size=<max_queue_size> --port=<port> --source=<source>
           --gitsource=<gitdsourceurl>
           --oauthserver=<oserv> --target=<target> --storeURL=<url> --mServer=<server> --mFrom=<from>
           --doiMaster=<master> --doiBase=<base> --sendMail --debug --templates=<tmplpath> --pubkey=<key>]

Options:
  --max_workers=<max_workers>     The number of workers to start [default: 3]
  --max_queue_size=<max_quesize>  The The size of the job queue [default: 100]
  --port=<port>                   The server port [default: 8083]
  --source=<dsourceurl>           The Server adress from which data can be read [default: https://repo.gin.g-node.org/]
  --gitsource=<gitdsourceurl>     The git Server adress from which data can be cloned [default: git@gin.g-node.org:]
  --oauthserver=<repo>            The Server aof the repo service [default: https://auth.gin.g-node.org]
  --target=<target>               The Location for long term storgae [default: data]
  --storeURL=<url>                The base url for storage [default: http://doid.gin.g-node.org/]
  --mServer=<server>              The mailserver adress (:and port) [default: localhost:25]
  --mFrom=<from>                  The mail from adress [default: no-reply@g-node.org]
  --doiMaster=<master>            The mail adress to send info to [default: dev@g-node.org]
  --doiBase=<base>                The first part of the DOI [default: 10.12751]
  --sendMail                      Whether Mail Noticiations should really be send (Otherwise just print them)
  --debug                         Whether debug messages shall be printed
  --templates=<tmplpath>          Path to the Templates [default: tmpl]
  --pubkey=<key>		  Path to the ssh Public Key of the doi user [default: ~/.ssh/id_rsa.pub]
 `

	args, err := docopt.Parse(usage, nil, true, "gin doi 0.1a", false)
	if err != nil {
		log.Printf("Error while parsing command line: %+v", err)
		os.Exit(-1)
	}
	// Setup data source
	ds := &ginDoi.GinDataSource{GinURL: args["--source"].(string), GinGitURL: args["--gitsource"].(string)}

	// doi provider
	dp := ginDoi.GnodeDoiProvider{ApiURI: "", DOIBase: args["--doiBase"].(string)}

	//Setup storage
	mServer := ginDoi.MailServer{Adress: args["--mServer"].(string), From: args["--mFrom"].(string),
		DoSend:                      args["--sendMail"].(bool),
		Master:                      args["--doiMaster"].(string)}
	storage := ginDoi.LocalStorage{Path: args["--target"].(string), Source: ds, HttpBase: args["--storeURL"].(string),
		DProvider:                   dp, MServer: &mServer, TemplatePath: args["--templates"].(string)}

	// setup authentication
	oaAdress := args["--oauthserver"].(string)
	op := ginDoi.GinOauthProvider{Uri: fmt.Sprintf("%s/api/accounts/", oaAdress),
		TokenURL:                  fmt.Sprintf("%s/oauth/validate/\\%s", oaAdress)}

	// Create the job queue.
	maxQ, err := strconv.Atoi(args["--max_queue_size"].(string))
	if err != nil {
		log.Printf("Error while parsing command line: %+v", err)
		os.Exit(-1)
	}
	jobQueue := make(chan ginDoi.Job, maxQ)
	// Start the dispatcher.
	maxW, err := strconv.Atoi(args["--max_workers"].(string))
	dispatcher := ginDoi.NewDispatcher(jobQueue, maxW)
	dispatcher.Run(ginDoi.NewWorker)

	//get the doi users ssh key
	fp, err := os.Open(args["--pubkey"].(string))
	if err != nil {
		log.Errorf("Could not open key file: %+v", err)
		os.Exit(-1)
	}
	key := gin.SSHKey{}
	RKey, err := ioutil.ReadAll(fp)
	if err != nil {
		log.Errorf("Could not read from key file: %+v", err)
		os.Exit(-1)
	}
	err = json.Unmarshal(RKey, &key)
	if err != nil {
		log.Errorf("Could not unmarshal key file: %+v", err)
		os.Exit(-1)
	}

	// Start the HTTP handlers.
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		ginDoi.InitDoiJob(w, r, ds, &op, storage.TemplatePath)
	})
	http.HandleFunc("/do/", func(w http.ResponseWriter, r *http.Request) {
		ginDoi.DoDoiJob(w, r, jobQueue, storage, &op, key)
	})
	http.Handle("/assets/",
		http.StripPrefix("/assets/", http.FileServer(http.Dir("/assets"))))

	//Debugging?
	if args["--debug"].(bool) {
		log.SetLevel(log.DebugLevel)
		log.SetFormatter(&log.TextFormatter{ForceColors: true})
	}


	log.Fatal(http.ListenAndServe(":"+args["--port"].(string), nil))
}
