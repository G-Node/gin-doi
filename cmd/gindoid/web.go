package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"io/ioutil"
	"net/http"

	log "github.com/sirupsen/logrus"
)

const (
	msgInvalidRequest    = `Invalid request data received.  Please note that requests should only be submitted through repository pages on <a href="https://gin.g-node.org">GIN</a>.  If you followed the instructions in the <a href="https://gin.g-node.org/G-Node/Info/wiki/DOIfile">DOI registration guide</a> and arrived at this error page, please <a href="mailto:gin@g-node.org">contact us</a> for assistance.`
	msgInvalidDOI        = `The DOI file was not valid. Please see <a href="https://gin.g-node.org/G-Node/Info/wiki/DOIfile">the DOI guide</a> for detailed instructions. `
	msgInvalidURI        = "Please provide a valid repository URI"
	msgAlreadyRegistered = `<i class="info icon"></i>
						<div class="content">
							<div class="header"> A DOI is already registered for your dataset.</div>
							Your DOI is: <br>
								<div class ="ui label label-default"><a href="https://doi.org/%s">%s</a>
							</div>.
						</div>`
	msgServerIsArchiving = `<div class="content">
			<div class="header">The DOI server has started archiving your repository.</div>
		We have reserved the following DOI for your dataset:<br>
		<div class ="ui label label-default">%s</div><br>
		In rare cases the final DOI might be different.<br>
		Please note that the final step in the registration process requires us to manually review your request.
		It may therefore take a few hours until the DOI is finally registered and your data becomes available.
		We will notify you via email once the process is finished.<br>
		<div class="ui tabs divider"> </div>
		<b>This page can safely be closed. You do not need to keep it open.</b>
		</div>
		`
	msgNotLoggedIn      = `You are not logged in with the gin service. Login <a href="http://gin.g-node.org/">here</a>`
	msgNoToken          = "No authentication token provided"
	msgNoUser           = "No username provided"
	msgNoTitle          = "No title provided."
	msgNoAuthors        = "No authors provided."
	msgInvalidAuthors   = "Not all authors valid. Please provide at least a last name and a first name."
	msgNoDescription    = "No description provided."
	msgNoLicense        = "No valid license provided. Please specify URL and name."
	msgInvalidReference = "A specified Reference is not valid. Please provide the name and type of the reference."
	msgBadEncoding      = `There was an issue with the content of the DOI file (datacite.yml). This might mean that the encoding is wrong. Please see <a href="https://gin.g-node.org/G-Node/Info/wiki/DOIfile">the DOI guide</a> for detailed instructions or contact gin@g-node.org for assistance.`

	msgSubmitError  = "An internal error occurred while we were processing your request.  The G-Node team has been notified of the problem and will attempt to repair it and process your request.  We may contact you for further information regarding your request.  Feel free to <a href=mailto:gin@g-node.org>contact us</a> if you would like to provide more information or ask about the status of your request."
	msgSubmitFailed = "An internal error occurred while we were processing your request.  Your request was not submitted.  Please <a href=mailto:gin@g-node.org>contact us</a> for further assistance."
	// Log Prefixes
	lpAuth    = "GinOAP"
	lpStorage = "Storage"
	lpMakeXML = "MakeXML"
)

// startDOIRegistration starts the DOI registration process by authenticating
// with the GIN server and adding a new DOIJob to the jobQueue.
func startDOIRegistration(w http.ResponseWriter, r *http.Request, jobQueue chan DOIJob, conf *Configuration) {
	// Make sure we can only be called with an HTTP POST request.
	if r.Method != "POST" {
		w.Header().Set("Allow", "POST")
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	dReq := DOIReq{}

	// Send email notification even if the registration fails beyond this point
	defer notifyAdmin(&dReq, conf)

	// NOTE: If this function responds with an error header, the same error is
	// rendered regardless of the type. The error is rendered by the inline
	// 'doify' function in the requestPageTmpl template.

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		dReq.ErrorMessages = []string{"Failed to read request body. No info available."}
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	err = json.Unmarshal(body, &dReq)
	if err != nil {
		dReq.ErrorMessages = []string{"Failed to unmarshal request body. Some information may be missing."}
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	log.WithFields(log.Fields{
		"request": fmt.Sprintf("%+v", dReq),
		"source":  "startDOIRegistration",
	}).Debug("Received DOI request")

	// verify again
	if !verifyRequest(dReq.Repository, dReq.Username, dReq.Verification, conf.Key) {
		log.WithFields(log.Fields{
			"request": fmt.Sprintf("%+v", dReq),
			"source":  "startDOIRegistration",
		}).Error("Invalid request: failed to verify")
		w.WriteHeader(http.StatusUnauthorized)
		dReq.ErrorMessages = []string{"Failed to verify request"}
		return
	}

	user, err := conf.GIN.Session.RequestAccount(dReq.Username)
	if err != nil {
		log.WithFields(log.Fields{
			"request": fmt.Sprintf("%+v", dReq),
			"source":  "startDOIRegistration",
			"error":   err,
		}).Debug("Could not get userdata")
		w.WriteHeader(http.StatusUnauthorized)
		dReq.ErrorMessages = []string{fmt.Sprintf("Failed to get user data: %s", err.Error())}
		return
	}
	uuid := makeUUID(dReq.Repository)
	infoyml, err := readFileAtURL(dataciteURL(dReq.Repository, conf))
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		dReq.ErrorMessages = []string{fmt.Sprintf("Failed to fetch datacite.yml: %s", err.Error())}
		return
	}
	doiInfo, err := parseDOIInfo(infoyml)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		dReq.ErrorMessages = []string{fmt.Sprintf("Failed to parse datacite.yml: %s", err.Error())}
		return
	}

	doiInfo.UUID = uuid
	doi := conf.DOIBase + doiInfo.UUID[:6]
	doiInfo.DOI = doi
	dReq.DOIInfo = doiInfo

	if isRegisteredDOI(doi) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(fmt.Sprintf(msgAlreadyRegistered, doi, doi)))
		dReq.ErrorMessages = []string{"This DOI is already registered"}
		return
	}

	// Add job to queue
	job := DOIJob{Source: dReq.Repository, User: user, Request: dReq, Name: doiInfo.DOI, Config: conf}
	jobQueue <- job
	// Render success
	w.WriteHeader(http.StatusCreated)
	w.Write([]byte(fmt.Sprintf(msgServerIsArchiving, doi)))
}

// renderRequestPage renders the page for the staging area, where information
// is provided to the user and offers to start the DOI registration request.
// It validates the metadata provided from the GIN repository and shows
// appropriate error messages and instructions.
func renderRequestPage(w http.ResponseWriter, r *http.Request, conf *Configuration) {
	log.Infof("Got a new DOI request")
	if err := r.ParseForm(); err != nil {
		log.WithFields(log.Fields{
			"source": "Init",
		}).Debug("Could not parse form data")
		w.WriteHeader(http.StatusInternalServerError)
		// TODO: Notify via email (maybe)
		return
	}
	tmpl, err := template.New("requestpage").Parse(requestPageTmpl)
	if err != nil {
		log.WithFields(log.Fields{
			"source": "startDOIRegistration",
			"error":  err,
		}).Debug("Could not parse init template")
		w.WriteHeader(http.StatusInternalServerError)
		// TODO: Notify via email
		return
	}

	repository := r.Form.Get("repo")
	verification := r.Form.Get("verification")
	username := r.Form.Get("user")

	log.Infof("Got request: [repository: %s] [username: %s] [verification: %s]", repository, username, verification)
	dReq := DOIReq{Username: username, Repository: repository, Verification: verification}
	dReq.DOIInfo = &DOIRegInfo{}

	// If all are missing, redirect to root path?

	// If any of the values is missing, render invalid request page
	if len(repository) == 0 || len(username) == 0 || len(verification) == 0 {
		log.WithFields(log.Fields{
			"source":       "renderRequestPage",
			"repository":   repository,
			"username":     username,
			"verification": verification,
		}).Error("Invalid request: missing fields in query string")
		w.WriteHeader(http.StatusBadRequest)
		dReq.Message = template.HTML(msgInvalidRequest)
		tmpl.Execute(w, dReq)
		return
	}

	// Check verification string
	if !verifyRequest(repository, username, verification, conf.Key) {
		log.WithFields(log.Fields{
			"source":       "renderRequestPage",
			"repository":   repository,
			"username":     username,
			"verification": verification,
		}).Error("Invalid request: failed to verify")
		w.WriteHeader(http.StatusBadRequest)
		dReq.Message = template.HTML(msgInvalidRequest)
		tmpl.Execute(w, dReq)
		return
	}

	infoyml, err := readFileAtURL(dataciteURL(dReq.Repository, conf))
	if doiInfo, err := parseDOIInfo(infoyml); err == nil {
		// TODO: Simplify this chain of conditions
		j, _ := json.MarshalIndent(doiInfo, "", "  ")
		log.Debugf("Received DOI information: %s", string(j))
		dReq.DOIInfo = doiInfo
		err = tmpl.Execute(w, dReq)
		if err != nil {
			log.WithFields(log.Fields{
				"request": dReq,
				"source":  "Init",
				"error":   err,
			}).Error("Could not parse template")
			return
		}
	} else if doiInfo != nil {
		log.WithFields(log.Fields{
			"doiInfo": doiInfo,
			"source":  "Init",
			"error":   err,
		}).Debug("DOI file invalid")
		if doiInfo.Missing != nil {
			dReq.Message = template.HTML(msgInvalidDOI + " <p>Issue:<i> " + doiInfo.Missing[0] + "</i>")
		} else {
			dReq.Message = template.HTML(msgInvalidDOI + msgBadEncoding)
		}
		dReq.DOIInfo = &DOIRegInfo{}
		err = tmpl.Execute(w, dReq)
		if err != nil {
			log.WithFields(log.Fields{
				"doiInfo": doiInfo,
				"request": dReq,
				"source":  "Init",
				"error":   err,
			}).Error("Could not parse template")
			return
		}
		return
	} else {
		dReq.Message = template.HTML(msgInvalidDOI)
		tmpl.Execute(w, dReq)
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

// verifyRequest decrypts the verification string and compares it with the
// received data.
func verifyRequest(repo, username, verification, key string) bool {
	plaintext, err := decrypt([]byte(key), verification)
	if err != nil {
		log.WithFields(log.Fields{
			"source":       "verifyRequest",
			"repo":         repo,
			"username":     username,
			"verification": verification,
		}).Error("Invalid request: failed to decrypt verification string")
		return false
	}

	return plaintext == repo+username
}
