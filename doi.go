package main

import (
	"flag"
	"net/http"
	"log"
	"github.com/G-Node/gin-doi/src"
)

func main() {
	var (
		maxWorkers   = flag.Int("max_workers", 5, "The number of workers to start")
		maxQueueSize = flag.Int("max_queue_size", 100, "The size of job queue")
		port         = flag.String("port", "8083", "The server port")
		source       = flag.String("source", "https://repo.gin.g-node.org", "The default URI")
		baseTarget   = flag.String("target", "data/", "The default base path for storgae")
		httpStorrage   = flag.String("store", "http://doid.gin.g-node.org/", "The default base path for the external data store")
	)
	flag.Parse()
	ds := ginDoi.GinDataSource{GinURL: *source}
	dp := ginDoi.DoiProvider{ApiURI:"", DOIBase:"10.12751"}
	storage := ginDoi.LocalStorage{Path:*baseTarget, Source:ds, HttpBase:*httpStorrage,
					DProvider:dp}
	op := ginDoi.OauthProvider{Uri:"https://auth.gin.g-node.org/api/accounts"}
	// Create the job queue.
	jobQueue := make(chan ginDoi.Job, *maxQueueSize)
	// Start the dispatcher.
	dispatcher := ginDoi.NewDispatcher(jobQueue, *maxWorkers)
	dispatcher.Run(ginDoi.NewWorker)

	// Start the HTTP handler.
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		ginDoi.InitDoiJob(w, r, &ds, &op)
	})
	http.HandleFunc("/do/", func(w http.ResponseWriter, r *http.Request) {
		ginDoi.DoDoiJob(w,r,jobQueue, storage)
	})
	http.Handle("/assets/",
		http.StripPrefix("/assets/", http.FileServer(http.Dir("/assets"))))

	log.Fatal(http.ListenAndServe(":"+*port, nil))
}

