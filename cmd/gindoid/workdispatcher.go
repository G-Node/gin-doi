// Provides a simple Job Que and dispatching system. It is based on a blog post
// (http://marcio.io/2015/07/handling-1-million-requests-per-minute-with-golang/)
// The dispatching is kept (could be removed see
// https://gist.github.com/harlow/dbcd639cf8d396a2ab73) but as we might move to
// more advanced cross entity dispatching its still here
package main

import (
	"crypto/rsa"
	_ "expvar"
	_ "net/http/pprof"

	gogs "github.com/gogits/go-gogs-client"
	log "github.com/sirupsen/logrus"
)

// DOIJob holds the attributes needed to perform unit of work.
type DOIJob struct {
	Name    string
	Source  string
	User    gogs.User
	Request DOIReq
	Key     rsa.PrivateKey
	Config  *Configuration
}

// newWorker creates takes a numeric id and a channel w/ worker pool.
func newWorker(id int, workerPool chan chan DOIJob) Worker {
	return Worker{
		ID:         id,
		JobQueue:   make(chan DOIJob),
		WorkerPool: workerPool,
		QuitChan:   make(chan bool),
	}
}

type Worker struct {
	ID         int
	JobQueue   chan DOIJob
	WorkerPool chan chan DOIJob
	QuitChan   chan bool
}

func (w *Worker) start() {
	go func() {
		for {
			// Add my jobQueue to the worker pool.
			w.WorkerPool <- w.JobQueue
			select {
			case job := <-w.JobQueue:
				// Dispatcher has added a job to my jobQueue.
				createRegisteredDataset(job)
				log.WithFields(log.Fields{
					"source": "Worker",
				}).Debugf("Worker %d Completed %s!\n", w.ID, job.Name)
			case <-w.QuitChan:
				// We have been asked to stop.
				return
			}
		}
	}()
}

func (w *Worker) stop() {
	go func() {
		w.QuitChan <- true
	}()
}

// newDispatcher creates, and returns a new Dispatcher object.
func newDispatcher(jobQueue chan DOIJob, maxWorkers int) *Dispatcher {
	workerPool := make(chan chan DOIJob, maxWorkers)

	return &Dispatcher{
		jobQueue:   jobQueue,
		maxWorkers: maxWorkers,
		workerPool: workerPool,
	}
}

type Dispatcher struct {
	workerPool chan chan DOIJob
	maxWorkers int
	jobQueue   chan DOIJob
}

func (d *Dispatcher) run(makeWorker func(int, chan chan DOIJob) Worker) {
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
				log.WithFields(log.Fields{"jobname": job.Name}).
					Infof("Fetching workerJobQueue for: %s\n", job.Name)
				workerJobQueue := <-d.workerPool
				log.WithFields(log.Fields{"jobname": job.Name}).
					Infof("Adding %s to workerJobQueue\n", job.Name)
				workerJobQueue <- job
			}()
		}
	}
}
