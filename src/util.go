package ginDoi

import (
	"encoding/json"
	"fmt"
	log "github.com/Sirupsen/logrus"
	"html/template"
	"io/ioutil"
	"net/http"
	"path/filepath"
)



// Check the current user. Return a user if logged in
func loggedInUser(r *http.Request, pr *OauthProvider) (*DoiUser, error) {
	return &DoiUser{}, nil
}

func readBody(r *http.Request) (*string, error) {
	body, err := ioutil.ReadAll(r.Body)
	x := string(body)
	return &x, err
}

func DoDoiJob(w http.ResponseWriter, r *http.Request, jobQueue chan DoiJob, storage LocalStorage, op OauthProvider) {
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
		"request": fmt.Sprintf("%+v", dReq),
		"source":  "DoDoiJob",
	}).Debug("Unmarshaled a doi request")

	ok, err := op.ValidateToken(dReq.OauthLogin, dReq.Token)
	if err != nil {
		log.WithFields(log.Fields{
			"request": fmt.Sprintf("%+v", dReq),
			"source":  "DoDoiJob",
			"error":   err,
		}).Debug("User authentication Failed")
		dReq.Mess = MS_NOLOGIN
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	if ! ok {
		log.WithFields(log.Fields{
			"request": fmt.Sprintf("%+v", dReq),
			"source":  "DoDoiJob",
		}).Debug("Token not valid")
		dReq.Mess = MS_NOLOGIN
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	user, err := op.getUser(dReq.OauthLogin, dReq.Token)
	if err != nil {
		log.WithFields(log.Fields{
			"request": fmt.Sprintf("%+v", dReq),
			"source":  "DoDoiJob",
			"error":   err,
		}).Debug("Could not get userdata")
		dReq.Mess = MS_NOLOGIN
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	dReq.User = DoiUser{MainOId: user}
	//ToDo Error checking
	ds, _ := storage.GetDataSource()
	uuid, _ := ds.MakeUUID(dReq.URI, user)
	if ok, doiInfo := ds.ValidDoiFile(dReq.URI, user); !ok {
		w.WriteHeader(http.StatusBadRequest)
		return
	} else {
		doiInfo.UUID = uuid
		doi := storage.DProvider.MakeDoi(doiInfo)
		dReq.DoiInfo = *doiInfo
		key, err := op.AuthorizePull(user)
		if err != nil {
			log.WithFields(log.Fields{
				"source":  "DoDoiJob",
				"error":   err,
			}).Error("Could not Authorize Pull")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		job := DoiJob{Source: dReq.URI, Storage: storage, User: user, DoiReq: dReq, Name: doiInfo.UUID, Key: *key}
		jobQueue <- job
		// Render success.
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(fmt.Sprintf(MS_SERVERWORKS, storage.HttpBase+uuid, doi)))
	}
}

func InitDoiJob(w http.ResponseWriter, r *http.Request, ds DataSource, op OauthProvider,
	tp string) {
	log.Infof("Got a new DOI request")
	if err := r.ParseForm(); err != nil {
		log.WithFields(log.Fields{
			"source": "Init",
		}).Debug("Could not parse form data")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	URI := r.Form.Get("repo")
	token := r.Form.Get("token")
	username := r.Form.Get("user")
	dReq := DoiReq{URI: URI, OauthLogin: username, Token: token}
	log.WithFields(log.Fields{
		"request": fmt.Sprintf("%+v", dReq),
		"source":  "Init",
	}).Debug("Got DOI Request")

	t, err := template.ParseFiles(filepath.Join(tp, "initjob.html")) // Parse template file.
	if err != nil {
		log.WithFields(log.Fields{
			"source": "DoDoiJob",
			"error":  err,
		}).Debug("Could not parse init template")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// Test whether URi was provided
	if !(len(URI) > 0) {
		log.WithFields(log.Fields{
			"request": dReq,
			"source":  "Init",
			"error":   err,
		}).Debug("No Repo URI provided")
		dReq.Mess = MS_URIINVALID
		err := t.Execute(w, dReq)
		if err != nil {
			log.WithFields(log.Fields{
				"request": dReq,
				"source":  "Init",
				"error":   err,
			}).Debug("Template not parsed")
			return
		}
		return
	}

	// Test whether token was provided
	if !(len(token) > 0) {
		dReq.Mess = MS_NOTOKEN
		log.WithFields(log.Fields{
			"request": dReq,
			"source":  "Init",
			"error":   err,
		}).Debug("No Token provided")
		err := t.Execute(w, dReq)
		if err != nil {
			log.Print(err)
			return
		}
		return
	}

	// Test whether username was provided
	if !(len(username) > 0) {
		dReq.Mess = MS_NOUSER
		err := t.Execute(w, dReq)
		if err != nil {
			log.Print(err)
			return
		}
		return
	}

	// test user login
	ok, err := op.ValidateToken(username, token)
	if err != nil {
		log.WithFields(log.Fields{
			"request": fmt.Sprintf("%+v", dReq),
			"source":  "InitDoiJob",
			"error":   err,
		}).Debug("User authentication Failed")
		dReq.Mess = MS_NOLOGIN
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	if ! ok {
		log.WithFields(log.Fields{
			"request": fmt.Sprintf("%+v", dReq),
			"source":  "InitDoiJob",
		}).Debug("Token not valid")
		dReq.Mess = MS_NOLOGIN
		w.WriteHeader(http.StatusUnauthorized)
		t.Execute(w, dReq)
		return
	}

	// get user
	user, err := op.getUser(username, token)
	if err != nil {
		log.WithFields(log.Fields{
			"request": dReq,
			"source":  "Init",
			"error":   err,
		}).Debug("Could not authenticate user")
		dReq.Mess = MS_NOLOGIN
		t.Execute(w, dReq)
		return
	}

	// check for doifile
	if ok, doiInfo := ds.ValidDoiFile(URI, user); ok {
		log.WithFields(log.Fields{
			"doiInfo": doiInfo,
			"source":  "Init",
		}).Debug("Received Doi information")
		dReq.DoiInfo = *doiInfo
		err := t.Execute(w, dReq)
		if err != nil {
			log.WithFields(log.Fields{
				"request": dReq,
				"source":  "Init",
				"error":   err,
			}).Error("Could not parse template")
			return
		}
	} else {
		log.WithFields(log.Fields{
			"doiInfo": doiInfo,
			"source":  "Init",
			"error":   err,
		}).Debug("Cloudberry File invalid")
		dReq.Mess = MS_INVALIDDOIFILE + " Issue: " + doiInfo.Missing[0]
		t.Execute(w, dReq)
		if err != nil {
			log.WithFields(log.Fields{
				"request": dReq,
				"source":  "Init",
				"error":   err,
			}).Error("Could not parse template")
			return
		}
		return
	}
}
