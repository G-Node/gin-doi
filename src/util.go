package ginDoi

import (
	"net/http"
	"fmt"
)

// Responsible for storing smth defined by source to a kind of Storage 
// defined by target
type StorageElement interface {
	// Should return true if the target location is alredy there
	Exists(target string) (bool, error)
	// Store the things specifies by source in target  
	Put(source string, target string) (bool, error)
	GetDataSource() (*GinDataSource, error)
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

func RequestHandler(w http.ResponseWriter, r *http.Request, jobQueue chan Job, storage LocalStorage) {
	// Make sure we can only be called with an HTTP POST request.
	if r.Method != "POST" {
		w.Header().Set("Allow", "POST")
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	
	user, err := loggedInUser(r, &OauthProvider{})
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		return 
	}
	
	URI,err := readBody(r)
	//ToDo Error checking
	ds,_ := storage.GetDataSource()
	df,_ := ds.GetDoiFile(URI)
	if ok,doiInfo := validDoiFile(df); !ok {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Print(doiInfo)
		return 
	}

	// Create Job and push the work onto the jobQueue.
	job := Job{Source:*URI, Storage:storage, User: *user}
	jobQueue <- job

	// Render success.
	w.WriteHeader(http.StatusCreated)
}

