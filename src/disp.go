// Disp provides a simple Job Que and dispatching system. It is based on a blog post
// (http://marcio.io/2015/07/handling-1-million-requests-per-minute-with-golang/)
// The dispatching is kept (coudl be removed see https://gist.github.com/harlow/dbcd639cf8d396a2ab73)
// but as we might move to more advanced cross entity dispatching its still here
package ginDoi

import (
	_ "expvar"
	_ "net/http/pprof"
	log "github.com/Sirupsen/logrus"
)

// NewWorker creates takes a numeric id and a channel w/ worker pool.
func NewWorker(id int, workerPool chan chan Job) Worker {
	return Worker{
		Id:         id,
		JobQueue:   make(chan Job),
		WorkerPool: workerPool,
		QuitChan:   make(chan bool),
	}
}

type Worker struct {
	Id         int
	JobQueue   chan Job
	WorkerPool chan chan Job
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
				job.Storage.Put(job.Source, job.Name, &job.DoiReq)
				log.WithFields(log.Fields{
					"source": "Worker",
				}).Debugf("Worker %d Completed %s!\n", w.Id, job.Name)
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

// NewDispatcher creates, and returns a new Dispatcher object.
func NewDispatcher(jobQueue chan Job, maxWorkers int) *Dispatcher {
	workerPool := make(chan chan Job, maxWorkers)

	return &Dispatcher{
		jobQueue:   jobQueue,
		maxWorkers: maxWorkers,
		workerPool: workerPool,
	}
}

type Dispatcher struct {
	workerPool chan chan Job
	maxWorkers int
	jobQueue   chan Job
}

func (d *Dispatcher) Run(makeWorker func(int, chan chan Job)Worker) {
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
				log.WithFields(log.Fields{"jobname":job.Name}).
					Infof("Fetching workerJobQueue for: %s\n", job.Name)
				workerJobQueue := <-d.workerPool
				log.WithFields(log.Fields{"jobname":job.Name}).
					Infof("Adding %s to workerJobQueue\n", job.Name)
				workerJobQueue <- job
			}()
		}
	}
}