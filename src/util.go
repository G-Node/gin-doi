package ginDoi

import (
	"net/http"
	"html/template"
	"fmt"
	"log"
	"io/ioutil"
)

// Job holds the attributes needed to perform unit of work.
type Job struct {
	Name  string
	Source string
	Storage LocalStorage
	User DoiUser
	DoiInfo DoiInfo
}

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
	return &DoiUser{}, nil
}


func readBody(r *http.Request) (*string, error){
	body, err := ioutil.ReadAll(r.Body)
	x:= string(body)
	return &x, err
}

func DoDoiJob(w http.ResponseWriter, r *http.Request, jobQueue chan Job, storage LocalStorage) {
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
	log.Printf("Git URI:%s", *URI)
	//ToDo Error checking
	ds,_ := storage.GetDataSource()
	df,_ := ds.GetDoiFile(*URI)
	uuid, _ := ds.MakeUUID(*URI)
	if ok,doiInfo := validDoiFile(df); !ok {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Print(doiInfo)
		return 
	}else{
		job := Job{Source:*URI, Storage:storage, User: *user, DoiInfo:doiInfo, Name:uuid}
		jobQueue <- job
		// Render success.
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte("The Doi Server has started doifying you repository. Once finnished it will be availible under: %s. Please return to that location to check for availibility"))
	}
}

type DoiAnswer struct {
	DoiInfo DoiInfo
	Mess string
	URI string
}
func InitDoiJob(w http.ResponseWriter, r *http.Request, ds *GinDataSource) {
	if err := r.ParseForm(); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	URI := r.Form.Get("repo")
	log.Printf("Got Body text:%s",URI)
	t, err := template.ParseFiles("tmpl/initjob.html") // Parse template file.
	if err != nil {
		log.Print(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if len(URI)>0 {
		doiI, err := ds.GetDoiFile(URI)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		if ok, doiInfo := validDoiFile(doiI); ok {
			err := t.Execute(w, DoiAnswer{doiInfo, "Hello",URI})
			if err != nil {
				log.Print(err)
				return
			}
		} else {
			log.Println(err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	} else{
		err := t.Execute(w, DoiAnswer{DoiInfo{}, "Please provide a repository URI", ""})
		if err != nil {
			log.Print(err)
			return
		}
	}
}


