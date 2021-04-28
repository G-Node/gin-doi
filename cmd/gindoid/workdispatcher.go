// Provides a simple Job Que and dispatching system. It is based on a blog post
// (http://marcio.io/2015/07/handling-1-million-requests-per-minute-with-golang/)
// The dispatching is kept (could be removed see
// https://gist.github.com/harlow/dbcd639cf8d396a2ab73) but as we might move to
// more advanced cross entity dispatching its still here
package main

import (
	_ "expvar"
	"log"
	_ "net/http/pprof"

	"github.com/G-Node/libgin/libgin"
)

// RegistrationJob holds a reference to the Metadata associated with a job and
// the service Configuration.
type RegistrationJob struct {
	Metadata *libgin.RepositoryMetadata
	Config   *Configuration
}

// newWorker creates a worker that waits for new jobs on its JobQueue starts a
// registration process when a job is received.
func newWorker(id int, workerPool chan chan *RegistrationJob) Worker {
	return Worker{
		ID:         id,
		JobQueue:   make(chan *RegistrationJob),
		WorkerPool: workerPool,
		QuitChan:   make(chan bool),
	}
}

// Worker holds a JobQueue that will accept and handle incoming
// DOI registration jobs.
type Worker struct {
	ID         int
	JobQueue   chan *RegistrationJob
	WorkerPool chan chan *RegistrationJob
	QuitChan   chan bool
}

// start the worker and wait for jobs.
func (w *Worker) start() {
	go func() {
		for {
			// Add my jobQueue to the worker pool.
			w.WorkerPool <- w.JobQueue
			select {
			case job := <-w.JobQueue:
				// Dispatcher has added a job to my jobQueue.
				createRegisteredDataset(job)
				log.Printf("Worker %d Completed %q!", w.ID, job.Metadata.SourceRepository)
			case <-w.QuitChan:
				// We have been asked to stop.
				return
			}
		}
	}()
}

// stop the worker.
func (w *Worker) stop() {
	go func() {
		w.QuitChan <- true
	}()
}

// newDispatcher creates and returns a new Dispatcher object that holds all
// waiting jobs and sends the next job in the queue to the first available
// worker.
func newDispatcher(jobQueue chan *RegistrationJob, maxWorkers int) *Dispatcher {
	workerPool := make(chan chan *RegistrationJob, maxWorkers)

	return &Dispatcher{
		jobQueue:   jobQueue,
		maxWorkers: maxWorkers,
		workerPool: workerPool,
	}
}

// Dispatcher holds waiting jobs and sends the next job in the queue to
// the first available worker.
type Dispatcher struct {
	workerPool chan chan *RegistrationJob
	maxWorkers int
	jobQueue   chan *RegistrationJob
}

// run starts the dispatcher after creating and starting a new set of workers
// (given the provided function and the predefined max workers).
func (d *Dispatcher) run(makeWorker func(int, chan chan *RegistrationJob) Worker) {
	for i := 0; i < d.maxWorkers; i++ {
		worker := makeWorker(i+1, d.workerPool)
		worker.start()
	}

	go d.dispatch()
}

func (d *Dispatcher) dispatch() {
	for {
		select {
		case job := <-d.jobQueue:
			go func() {
				log.Printf("Fetching workerJobQueue for %q", job.Metadata.SourceRepository)
				workerJobQueue := <-d.workerPool
				log.Printf("Adding %q to workerJobQueue", job.Metadata.SourceRepository)
				workerJobQueue <- job
			}()
		}
	}
}
