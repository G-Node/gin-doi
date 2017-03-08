package ginDoi

import (
	"net/http"
	"html/template"
	"fmt"
	"log"
	"io/ioutil"
	"encoding/json"
	"path/filepath"
)

var(
	MS_NODOIFILE = 		"Could no locte a Doi File. Please visit http://... for a guide"
	MS_INVALIDDOIFILE = 	"The doi File ws not Valid. Please visit http://... for a guide"
	MS_URIINVALID =   	"Please provide a valid repository URI"
	MS_SERVERWORKS = 	"The Doi Server has started doifying you repository. " +
				"Once finnished it will be availible at the location below Please return to that location to check for " +
				"availibility <br><br>"+
				"<a href=\"%s\" class=\"label label-warning\">Your Landing Page</a>"
	MS_NOLOGIN =		"You are not logged in with the gin service. Login at: http://gin.g-node.org/"

)

// Job holds the attributes needed to perform unit of work.
type Job struct {
	Name    string
	Source  string
	Storage LocalStorage
	User    OauthIdentity
	DoiReq  DoiReq
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
	FirstName string `json:"first_name"`
        LastName string `json:"last_name"`
        Token string
        EmailRaw json.RawMessage `json:"email"`
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

type DoiReq struct {
	URI string
	User string
	Token string
	Mess string
	DoiInfo CBerry
}

type CBerry struct {
	Missing []string
	DOI string
	UUID string
	FileSize int64
	Title string
	Authors []string
	Description string
	Keywords []string
	References string
	License string
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

func DoDoiJob(w http.ResponseWriter, r *http.Request, jobQueue chan Job, storage LocalStorage, op *OauthProvider) {
	// Make sure we can only be called with an HTTP POST request.
	if r.Method != "POST" {
		w.Header().Set("Allow", "POST")
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	dReq := DoiReq{}
	//ToDo Error checking
	body, _ := ioutil.ReadAll(r.Body)
	json.Unmarshal(body, &dReq)
	log.Printf("Git URI:%s", dReq.URI)

	user, err := op.getUser(dReq.User, dReq.Token)
	if err != nil{
		log.Printf("[Do doi Job]: Could not authenticate user %+v. Request Data: %+v", err, dReq)
		dReq.Mess = MS_NOLOGIN
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	//ToDo Error checking
	ds,_ := storage.GetDataSource()
	df,_ := ds.GetDoiFile(dReq.URI)
	uuid, _ := ds.MakeUUID(dReq.URI)

	if ok,doiInfo := validDoiFile(df); !ok {
		w.WriteHeader(http.StatusBadRequest)
		return 
	}else{
		doiInfo.UUID = uuid
		dReq.DoiInfo = *doiInfo
		job := Job{Source:dReq.URI, Storage:storage, User: user, DoiReq:dReq, Name:doiInfo.UUID}
		jobQueue <- job
		// Render success.
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(fmt.Sprintf(MS_SERVERWORKS, storage.HttpBase+uuid)))
	}
}

func InitDoiJob(w http.ResponseWriter, r *http.Request, ds *GinDataSource, op *OauthProvider) {
	if err := r.ParseForm(); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	URI := r.Form.Get("repo")
	token := r.Form.Get("token")
	username := r.Form.Get("user")
	dReq := DoiReq{URI:URI, User:username, Token:token}
	log.Printf("[Init] Repo: %+v", dReq)

	t, err := template.ParseFiles(filepath.Join("tmpl","initjob.html")) // Parse template file.
	if err != nil {
		log.Print(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	user, err := op.getUser(username, token)
	if err != nil{
		log.Printf("InitDoiJob: Could not authenticate user %v", err)
		dReq.Mess = MS_NOLOGIN
		t.Execute(w, dReq)
		return
	}

	log.Printf("[Init] User: %+v", user)


	if len(URI)>0 {
		doiI, err := ds.GetDoiFile(URI)
		if err != nil {
			log.Printf("InitDoiJob: Could not get Doi File %v", err)
			dReq.Mess = MS_NODOIFILE
			t.Execute(w, dReq)
			return
		}
		if ok, doiInfo := validDoiFile(doiI); ok {
			log.Printf("InitDoiJob: Received Doi information:%+v", doiInfo)
			dReq.DoiInfo = *doiInfo
			err := t.Execute(w, dReq)
			if err != nil {
				log.Printf("InitDoiJob: Could not parse template %v", err)
				return
			}
		} else {
			log.Printf("InitDoiJob: Cloudberry File invalid %v", err)
			dReq.Mess = MS_INVALIDDOIFILE
			t.Execute(w, dReq)
			return
		}
	} else{
		dReq.Mess = MS_URIINVALID
		err := t.Execute(w, dReq)
		if err != nil {
			log.Print(err)
			return
		}
	}
}


