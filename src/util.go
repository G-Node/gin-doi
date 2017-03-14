package ginDoi

import (
	"net/http"
	"html/template"
	"fmt"
	log "github.com/Sirupsen/logrus"
	"io/ioutil"
	"encoding/json"
	"path/filepath"
)

var(
	MS_NODOIFILE = 		"Could no locte a Doi File. Please visit https://web.gin.g-node.org/info/doi for a guide"
	MS_INVALIDDOIFILE = 	"The doi File ws not Valid. Please visit https://web.gin.g-node.org/info/doi for a guide"
	MS_URIINVALID =   	"Please provide a valid repository URI"
	MS_SERVERWORKS = 	"The Doi Server has started doifying you repository. " +
				"Once finnished it will be availible <a href=\"%s\" class=\"label label-warning\">here</a>. Please return to that location to check for " +
				"availibility <br><br>"+
				"We will try to resgister the follwoing doi: <div class =\"label label-default\">%s</div> " +
				"for your dataset. Please note, however, that in rare cases the final doi might be different."
	MS_NOLOGIN =		"You are not logged in with the gin service. Login at http://gin.g-node.org/"
	MS_NOTOKEN = 		"No authentication token provided"
	MS_NOUSER = 		"No username provided"

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
	log.WithFields(log.Fields{
		"request": dReq,
		"source": "DoDoiJob",
	}).Debug()

	user, err := op.getUser(dReq.User, dReq.Token)
	if err != nil{
		log.WithFields(log.Fields{
			"request": dReq,
			"source": "DoDoiJob",
			"error":err,
		}).Debug("Could not authenticate user")
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
		doi := storage.DProvider.MakeDoi(doiInfo)
		dReq.DoiInfo = *doiInfo
		job := Job{Source:dReq.URI, Storage:storage, User: user, DoiReq:dReq, Name:doiInfo.UUID}
		jobQueue <- job
		// Render success.
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(fmt.Sprintf(MS_SERVERWORKS, storage.HttpBase+uuid, doi)))
	}
}

func InitDoiJob(w http.ResponseWriter, r *http.Request, ds *GinDataSource, op *OauthProvider) {
	log.Infof("Got a new DOI request")
	if err := r.ParseForm(); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	URI := r.Form.Get("repo")
	token := r.Form.Get("token")
	username := r.Form.Get("user")
	dReq := DoiReq{URI:URI, User:username, Token:token}
	log.WithFields(log.Fields{
		"request": dReq,
		"source": "Init",
	}).Debug("Got DOI Request")
	log.Infof("Will Doify %s", dReq.URI)

	t, err := template.ParseFiles(filepath.Join("tmpl","initjob.html")) // Parse template file.
	if err != nil {
		log.WithFields(log.Fields{
			"request": dReq,
			"source": "DoDoiJob",
			"error":err,
		}).Debug("Could not parse init template")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// Test whether URi was provided
	if !(len(URI) > 0){
		log.WithFields(log.Fields{
			"request": dReq,
			"source": "Init",
			"error":err,
		}).Debug("No Repo URI provided")
		dReq.Mess = MS_URIINVALID
		err := t.Execute(w, dReq)
		if err != nil {
			log.WithFields(log.Fields{
				"request": dReq,
				"source": "Init",
				"error":err,
			}).Debug("Template not parsed")
			return
		}
		return
	}

	// Test whether token was provided
	if !(len(token) > 0){
		dReq.Mess = MS_NOTOKEN
		log.WithFields(log.Fields{
			"request": dReq,
			"source": "Init",
			"error":err,
		}).Debug("No Token provided")
		err := t.Execute(w, dReq)
		if err != nil {
			log.Print(err)
			return
		}
		return
	}

	// Test whether username was provided
	if !(len(username) > 0){
		dReq.Mess = MS_NOUSER
		err := t.Execute(w, dReq)
		if err != nil {
			log.Print(err)
			return
		}
		return
	}

	// test user login
	_, err = op.getUser(username, token)
	if err != nil{
		log.WithFields(log.Fields{
			"request": dReq,
			"source": "Init",
			"error":err,
		}).Debug("Could not authenticate user")
		dReq.Mess = MS_NOLOGIN
		t.Execute(w, dReq)
		return
	}

	doiI, err := ds.GetDoiFile(URI)
	if err != nil {
		log.WithFields(log.Fields{
			"request": dReq,
			"source": "Init",
			"error":err,
		}).Debug("Could not get Cloudberry File")
		dReq.Mess = MS_NODOIFILE
		t.Execute(w, dReq)
		return
	}

	if ok, doiInfo := validDoiFile(doiI); ok {
		log.WithFields(log.Fields{
			"doiInfo": doiInfo,
			"source": "Init",
		}).Debug("Received Doi information")
		dReq.DoiInfo = *doiInfo
		err := t.Execute(w, dReq)
		if err != nil {
			log.WithFields(log.Fields{
				"request": dReq,
				"source": "Init",
				"error":err,
			}).Error("Could not parse template")
			return
		}
	} else {
		log.WithFields(log.Fields{
			"doiInfo": doiInfo,
			"source": "Init",
			"error":err,
		}).Debug("Cloudberry File invalid")
		dReq.Mess = MS_INVALIDDOIFILE
		t.Execute(w, dReq)
		if err != nil {
			log.WithFields(log.Fields{
				"request": dReq,
				"source": "Init",
				"error":err,
			}).Error("Could not parse template")
			return
		}
		return
	}
}


