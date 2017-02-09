package ginDoi

import (
	"net/http"
	"flag"
	"log"
	"./disp"
	"fmt"
	"gopkg.in/yaml.v2"
)

// Responsible for storing smth defined by source to a kind of Storage 
// defined by target
type StorageElement interface {
	// Should return true if the target location is alredy there
	Exists(target string) (bool, error)
	// Store the things specifies by source in target  
	Put(source string, target string) (bool, error)
	GetDataSource() (DataSource, error)
}

type DataSource interface {
	GetDoiFile(URI string) ([]byte, error)
	Get(URI string) (string, error)
}

type OauthIdentity struct {
	Name string
	Mail string
	Token string
}

type OauthProvider struct {
	Name string
	Uri string
	ApiKey string
}

type DoiUser struct {
	Name string
	Identities []OauthIdentity
	MainOId OauthIdentity
}

type DoiInfo struct {
	URI string
	Title string
	Authors string
	Description string
	Keywords string
	References string
	License string
	Addendum string	
}


// Check the current user. Return a user if logged in
func loggedInUser(r *http.Request , pr *OauthProvider) (*DoiUser, error){
	return nil, nil
}


func readBody(r *http.Request) (*string, error){
	return nil, nil
}

type DOIJob struct {
	disp.Job
	Source  string
	Target  string
	Storage StorageElement
	User    DoiUser
}

type DOIWorker struct {
	disp.ExWorker
}

func (w *DOIWorker) start(){
	go func() {
		for {
			// Add my jobQueue to the worker pool.
			w.WorkerPool <- w.JobQueue
			select {
			case job := <-w.JobQueue:
			// Dispatcher has added a job to my jobQueue.
				fmt.Printf("worker%d: started %sn", w.Id, job.Name)
				x :=DOIJob(job)
				if ok,_:=x.Storage.Exists(x.Target);!ok {
					x.Storage.Put(x.Source, x.Target)
					w.sendMail(job)
				}
				fmt.Printf("worker%d: completed %s!\n", w.Id, job.Name)
			case <-w.QuitChan:
			// We have been asked to stop.
				fmt.Printf("worker%d stopping\n", w.Id)
				return
			}
		}
	}()
}

func (w *DOIWorker) sendMail(j DOIJob) error {
	fmt.Println("Would send Mail about job %s",j.Name)
	return nil
}

func requestHandler(w http.ResponseWriter, r *http.Request, jobQueue chan disp.Job, storage StorageElement) {
	// Make sure we can only be called with an HTTP POST request.
	if r.Method != "POST" {
		w.Header().Set("Allow", "POST")
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	
	user, err := loggedInUser(r, OauthProvider{})
	if !err {
		w.WriteHeader(http.StatusUnauthorized)
		return 
	}
	
	URI,err := readBody(r)
	//ToDo Error checking
	ds,_ := storage.GetDataSource()
	df,_ :=ds.GetDoiFile(URI)
	if !err || !validDoiFile(df){
		w.WriteHeader(http.StatusBadRequest)
		return 
	}
		
	// Create Job and push the work onto the jobQueue.
	job := DOIJob{Source:URI, Storage:storage, User: user}
	jobQueue <- job

	// Render success.
	w.WriteHeader(http.StatusCreated)
}

func main() {
	var (
		maxWorkers   = flag.Int("max_workers", 5, "The number of workers to start")
		maxQueueSize = flag.Int("max_queue_size", 100, "The size of job queue")
		port         = flag.String("port", "8080", "The server port")
	)
	flag.Parse()
	
	// Create the job queue.
	jobQueue := make(chan disp.Job, *maxQueueSize)

	// Start the dispatcher.
	dispatcher := disp.NewDispatcher(jobQueue, *maxWorkers)
	dispatcher.Run(disp.NewWorker)
	x := StorageElement{}
	// Start the HTTP handler.
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		requestHandler(w, r, jobQueue, x)
	})
	log.Fatal(http.ListenAndServe(":"+*port, nil))
}
