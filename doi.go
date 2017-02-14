package main

import (
	"flag"
	"net/http"
	"log"
	"./src"
)

func main() {
	var (
		maxWorkers   = flag.Int("max_workers", 5, "The number of workers to start")
		maxQueueSize = flag.Int("max_queue_size", 100, "The size of job queue")
		port         = flag.String("port", "8080", "The server port")
	)
	flag.Parse()

	// Create the job queue.
	jobQueue := make(chan ginDoi.Job, *maxQueueSize)

	// Start the dispatcher.
	dispatcher := ginDoi.NewDispatcher(jobQueue, *maxWorkers)
	dispatcher.Run(ginDoi.NewWorker)
	x := ginDoi.LocalStorage{}
	// Start the HTTP handler.
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		ginDoi.RequestHandler(w, r, jobQueue, x)
	})
	log.Fatal(http.ListenAndServe(":"+*port, nil))
}

