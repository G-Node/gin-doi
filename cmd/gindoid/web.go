package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"

	"github.com/G-Node/libgin/libgin"
)

const (
	msgInvalidRequest    = `Invalid request data received.  Please note that requests should only be submitted through repository pages on <a href="https://gin.g-node.org">GIN</a>.  If you followed the instructions in the <a href="https://gin.g-node.org/G-Node/Info/wiki/DOIfile">DOI registration guide</a> and arrived at this error page, please <a href="mailto:gin@g-node.org">contact us</a> for assistance.`
	msgInvalidDOI        = `The DOI file is missing or not valid. Please see <a href="https://gin.g-node.org/G-Node/Info/wiki/DOIfile">the DOI guide</a> for detailed instructions. `
	msgInvalidURI        = "Please provide a valid repository URI"
	msgAlreadyRegistered = `<div class="content">
								<div class="header"> A DOI is already registered for your dataset.</div>
								Your DOI is: <br>
								<div class ="ui label label-default"><a href="https://doi.org/%s">%s</a></div></br>
								If this is incorrect or you would like to register a new version of your dataset, please <a href=mailto:gin@g-node.org>contact us</a>.
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

	msgSubmitError     = "An internal error occurred while we were processing your request.  The G-Node team has been notified of the problem and will attempt to repair it and process your request.  We may contact you for further information regarding your request.  Feel free to <a href=mailto:gin@g-node.org>contact us</a> if you would like to provide more information or ask about the status of your request."
	msgSubmitFailed    = "An internal error occurred while we were processing your request.  Your request was not submitted and the service failed to notify the G-Node team.  Please <a href=mailto:gin@g-node.org>contact us</a> to report this error."
	msgNoTemplateError = "An internal error occurred while we were processing your request.  The G-Node team has been notified of the problem and will attempt to repair it and process your request.  We may contact you for further information regarding your request.  Feel free to contact us at gin@g-node.org if you would like to provide more information or ask about the status of your request."
	// Log Prefixes
	lpAuth    = "GinOAP"
	lpStorage = "Storage"
	lpMakeXML = "MakeXML"
)

type reqResultData struct {
	Success bool
	Level   string // success, warning, error
	Message template.HTML
	Request *DOIReq
}

// renderResult renders the results of a registration request using the
// 'requestResultTmpl' template. If it fails to parse the template, it renders
// the Message from the result data in plain HTML.
func renderResult(w http.ResponseWriter, resData *reqResultData) {
	tmpl, err := template.New("requestresult").Parse(requestResultTmpl)
	if err != nil {
		log.Printf("Failed to parse template: %s", err.Error())
		log.Printf("Request data: %+v", resData)
		// failed to render result template; just show the message wrapped in html tags
		w.Write([]byte("<html>" + resData.Message + "</html>"))
		return
	}
	tmpl.Execute(w, &resData)
}

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
	resData := reqResultData{Request: &dReq}

	dReq.RequestData = r.PostFormValue("reqdata")
	reqdata, err := decryptRequestData(dReq.RequestData, conf.Key)
	if err != nil {
		log.Printf("Invalid request: %s", err.Error())
		dReq.ErrorMessages = []string{"Failed to verify request"}
		resData.Message = template.HTML(msgInvalidRequest)
		// ignore the error, no email to send
		renderResult(w, &resData)
		return
	}

	dReq.DOIRequestData = reqdata

	log.Printf("Received DOI request: %+v", dReq)

	// calculate DOI
	uuid := makeUUID(dReq.Repository)
	doi := conf.DOIBase + uuid[:6]

	if isRegisteredDOI(doi) {
		resData.Success = false
		resData.Level = "warning"
		resData.Message = template.HTML(fmt.Sprintf(msgAlreadyRegistered, doi, doi))
		renderResult(w, &resData)
		return
	}

	// everything beyond this point should trigger an email notification
	defer func() {
		err := notifyAdmin(&dReq, conf)
		if err != nil {
			// Email send failed
			// Log the error
			log.Printf("Failed to send notification email: %s", err.Error())
			log.Printf("Request data: %+v", dReq)
			// Ask the user to contact us
			resData.Success = false
			resData.Level = "error"
			resData.Message = template.HTML(msgSubmitFailed)
		}
		// Render the result
		renderResult(w, &resData)
	}()

	user, err := conf.GIN.Session.RequestAccount(dReq.Username)
	if err != nil {
		// Can happen if the DOI service isn't logged in to GIN
		log.Printf("Failed to get user data: %s", err.Error())
		log.Printf("Request data: %+v", dReq)
		dReq.ErrorMessages = []string{fmt.Sprintf("Failed to get user data: %s", err.Error())}
		resData.Success = true
		resData.Level = "warning"
		resData.Message = template.HTML(msgSubmitError)
		return
	}
	infoyml, err := readFileAtURL(dataciteURL(dReq.Repository, conf))
	if err != nil {
		// Can happen if the datacite.yml file or the repository is removed (or
		// made private) between preparing the request and submitting it
		log.Printf("Failed to fetch datacite.yml: %s", err.Error())
		log.Printf("Request data: %+v", dReq)
		dReq.ErrorMessages = []string{fmt.Sprintf("Failed to fetch datacite.yml: %s", err.Error())}
		resData.Success = true
		resData.Level = "warning"
		resData.Message = template.HTML(msgSubmitError)
		return
	}
	doiInfo, err := parseDOIInfo(infoyml)
	if err != nil {
		// Can happen if the datacite.yml file is modified (and made invalid)
		// between preparing the request and submitting it
		log.Printf("Failed to parse datacite.yml: %s", err.Error())
		log.Printf("Request data: %+v", dReq)
		dReq.ErrorMessages = []string{fmt.Sprintf("Failed to parse datacite.yml: %s", err.Error())}
		resData.Success = true
		resData.Level = "warning"
		resData.Message = template.HTML(msgSubmitError)
		return
	}

	doiInfo.UUID = uuid
	doiInfo.DOI = doi
	dReq.DOIInfo = doiInfo

	// Add job to queue
	job := DOIJob{Source: dReq.Repository, User: user, Request: dReq, Name: doiInfo.DOI, Config: conf}
	jobQueue <- job
	// Render success
	message := fmt.Sprintf(msgServerIsArchiving, doi)
	resData.Success = true
	resData.Level = "success"
	resData.Message = template.HTML(message)
}

// renderRequestPage renders the page for the staging area, where information
// is provided to the user and offers to start the DOI registration request.
// It validates the metadata provided from the GIN repository and shows
// appropriate error messages and instructions.
func renderRequestPage(w http.ResponseWriter, r *http.Request, conf *Configuration) {
	log.Printf("Got a new DOI request")
	if err := r.ParseForm(); err != nil {
		log.Print("Could not parse form data")
		w.WriteHeader(http.StatusInternalServerError)
		// TODO: Notify via email (maybe)
		return
	}
	tmpl, err := template.New("requestpage").Parse(requestPageTmpl)
	if err != nil {
		log.Print("Could not parse init template")
		w.WriteHeader(http.StatusInternalServerError)
		// TODO: Notify via email
		return
	}

	regrequest := r.Form.Get("regrequest")

	log.Printf("Got request: %s", regrequest)

	dReq := DOIReq{}
	dReq.DOIInfo = &DOIRegInfo{}
	reqdata, err := decryptRequestData(regrequest, conf.Key)
	if err != nil {
		log.Printf("Invalid request: %s", err.Error())
		w.WriteHeader(http.StatusBadRequest)
		dReq.Message = template.HTML(msgInvalidRequest)
		tmpl.Execute(w, dReq)
		return
	}

	dReq.DOIRequestData = reqdata
	dReq.RequestData = regrequest // Forward it through the hidden form in the template

	infoyml, err := readFileAtURL(dataciteURL(dReq.Repository, conf))
	if err != nil {
		// Can happen if the datacite.yml file is removed and the user clicks DOIfy on a stale page
		log.Printf("Failed to fetch datacite.yml: %s", err.Error())
		log.Printf("Request data: %+v", dReq)
		dReq.ErrorMessages = []string{fmt.Sprintf("Failed to fetch datacite.yml: %s", err.Error())}
		dReq.Message = template.HTML(msgInvalidDOI + " <p>Issue: <i>No datacite.yml file found in repository</i>")
		err = tmpl.Execute(w, dReq)
		if err != nil {
			log.Print("Could not parse template")
		}
		return
	}
	if doiInfo, err := parseDOIInfo(infoyml); err == nil {
		// TODO: Simplify this chain of conditions
		j, _ := json.MarshalIndent(doiInfo, "", "  ")
		log.Printf("Received DOI information: %s", string(j))
		dReq.DOIInfo = doiInfo
		err = tmpl.Execute(w, dReq)
		if err != nil {
			log.Print("Could not parse template")
			return
		}
	} else if doiInfo != nil {
		log.Print("DOI file invalid")
		if doiInfo.Missing != nil {
			dReq.Message = template.HTML(msgInvalidDOI + " <p>Issue:<i> " + doiInfo.Missing[0] + "</i>")
		} else {
			dReq.Message = template.HTML(msgInvalidDOI + msgBadEncoding)
		}
		dReq.DOIInfo = &DOIRegInfo{}
		err = tmpl.Execute(w, dReq)
		if err != nil {
			log.Print("Could not parse template")
			return
		}
		return
	} else {
		dReq.Message = template.HTML(msgInvalidDOI)
		tmpl.Execute(w, dReq)
		if err != nil {
			log.Print("Could not parse template")
			return
		}
		return
	}
}

// decryptRequestData decrypts the submitted data into a map.  Returns with
// error if the decryption fails, the encrypted data is not a valid JSON
// object, or if any of the expected keys (username, realname, repository,
// email) are not present.
func decryptRequestData(regrequest string, key string) (*libgin.DOIRequestData, error) {
	plaintext, err := decrypt([]byte(key), regrequest)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt verification string: %s", err.Error())
	}

	data := libgin.DOIRequestData{}
	err = json.Unmarshal([]byte(plaintext), &data)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal request data: %s", err.Error())
	}

	// Required info: username, repo, email
	if data.Username == "" || data.Repository == "" || data.Email == "" {
		return nil, fmt.Errorf("invalid request: required key is missing or empty")
	}

	return &data, nil
}
