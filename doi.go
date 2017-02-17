package main

import (
	"flag"
	"net/http"
	"log"
	"./src"
	"fmt"
)

func main() {
	var (
		maxWorkers   = flag.Int("max_workers", 5, "The number of workers to start")
		maxQueueSize = flag.Int("max_queue_size", 100, "The size of job queue")
		port         = flag.String("port", "8081", "The server port")
	)
	flag.Parse()
	ds := ginDoi.GinDataSource{GinURL: "https://repo.gin.g-node.org"}
	// Create the job queue.
	jobQueue := make(chan ginDoi.Job, *maxQueueSize)

	storage := ginDoi.LocalStorage{Path:"./", Source:ds}
	// Start the dispatcher.
	dispatcher := ginDoi.NewDispatcher(jobQueue, *maxWorkers)
	dispatcher.Run(ginDoi.NewWorker)
	//x := ginDoi.LocalStorage{}
	// Start the HTTP handler.
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		ginDoi.InitDoiJob(w, r, &ds)
	})
	http.HandleFunc("/do/", func(w http.ResponseWriter, r *http.Request) {
		ginDoi.DoDoiJob(w,r,jobQueue, storage)
	})
	http.Handle("/assets/",
		http.StripPrefix("/assets/", http.FileServer(http.Dir("/assets"))))
	fmt.Print(maxWorkers)
	fmt.Print(maxQueueSize)
	fmt.Print(port)
	log.Fatal(http.ListenAndServe(":"+"8081", nil))
}

