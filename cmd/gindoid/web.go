package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"math/rand"
	"net/http"
	"strings"
	"time"

	gdtmpl "github.com/G-Node/gin-doi/templates"
	"github.com/G-Node/libgin/libgin"
	"github.com/spf13/cobra"
)

const (
	msgInvalidRequest    = `Invalid request data received.  Please note that requests should only be submitted through repository pages on <a href="https://gin.g-node.org">GIN</a>.  If you followed the instructions in the <a href="https://gin.g-node.org/G-Node/Info/wiki/DOIfile">DOI registration guide</a> and arrived at this error page, please <a href="mailto:gin@g-node.org">contact us</a> for assistance.`
	msgInvalidDOI        = `The DOI file is missing or not valid. See the messages below for specific issues with the provided data.<br>Also, please see <a href="https://gin.g-node.org/G-Node/Info/wiki/DOIfile">the DOI guide</a> for detailed instructions.`
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
	msgNoLicense        = "No valid license provided. Please specify a license URL and name and make sure it matches the license file in the repository."
	msgInvalidReference = "One of the Reference entries is not valid. Please provide the name and type of the reference."
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
	Success    bool
	Level      string // success, warning, error
	Message    template.HTML
	Repository string
}

// renderResult renders the results of a registration request using the
// 'RequestResult' template. If it fails to parse the template, it renders
// the Message from the result data in plain HTML.
func renderResult(w http.ResponseWriter, resData *reqResultData) {
	tmpl, err := template.New("requestresult").Parse(gdtmpl.RequestResult)
	if err != nil {
		log.Printf("Failed to parse requestresult template: %s", err.Error())
		log.Printf("Request data: %+v", resData)
		// failed to render result template; just show the message wrapped in html tags
		w.Write([]byte("<html>" + resData.Message + "</html>"))
		return
	}
	err = tmpl.Execute(w, &resData)
	if err != nil {
		log.Printf("Error rendering RequestResult template: %v", err.Error())
	}
}

const ALNUM = "1234567890abcdefghijklmnopqrstuvwxyz"

// randAlnum returns a random alphanumeric (lowercase, latin) string of length 'n'.
func randAlnum(n int) string {
	N := len(ALNUM)

	chrs := make([]byte, n)
	rand.Seed(time.Now().UnixNano())
	for idx := range chrs {
		chrs[idx] = ALNUM[rand.Intn(N)]
	}

	return string(chrs)
}

// startDOIRegistration starts the DOI registration process by authenticating
// with the GIN server and adding a new DOIJob to the jobQueue.
func startDOIRegistration(w http.ResponseWriter, r *http.Request, jobQueue chan *RegistrationJob, conf *Configuration) {
	// Make sure we can only be called with an HTTP POST request.
	if r.Method != "POST" {
		w.Header().Set("Allow", "POST")
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	errors := make([]string, 0, 5)

	regJob := &RegistrationJob{
		Metadata: new(libgin.RepositoryMetadata),
		Config:   conf,
	}
	resData := reqResultData{}

	encryptedRequestData := r.PostFormValue("reqdata")
	reqdata, err := decryptRequestData(encryptedRequestData, conf.Key)
	resData.Repository = reqdata.Repository
	if err != nil {
		log.Printf("Invalid request: %s", err.Error())
		resData.Message = template.HTML(msgInvalidRequest)
		// ignore the error, no email to send
		renderResult(w, &resData)
		return
	}

	log.Printf("Received DOI request: %+v", reqdata)

	requser := &libgin.GINUser{
		Username: reqdata.Username,
		RealName: reqdata.Realname,
		Email:    reqdata.Email,
	}
	regJob.Metadata.RequestingUser = requser
	regJob.Metadata.SourceRepository = reqdata.Repository

	// add fork repository to job data to render landing page
	repoParts := strings.SplitN(regJob.Metadata.SourceRepository, "/", 2)
	if len(repoParts) == 2 {
		regJob.Metadata.ForkRepository = strings.Join([]string{"doi", repoParts[1]}, "/")
	}
	// otherwise, unexpected repository name, so don't set ForkRepository and
	// the cloner will notify

	// exiting beyond this point should trigger an email notification
	defer func() {
		err := notifyAdmin(regJob, errors)
		if err != nil {
			// Email send failed
			// Log the error
			log.Printf("Failed to send notification email: %s", err.Error())
			log.Printf("Request data: %+v", reqdata)
			// Ask the user to contact us
			resData.Success = false
			resData.Level = "error"
			resData.Message = template.HTML(msgSubmitFailed)
		}
		// Render the result
		renderResult(w, &resData)
	}()

	// generate random DOI (keep generating if it's already registered)
	var doi string
	for ntry := 0; doi == "" && libgin.IsRegisteredDOI(doi); ntry++ {
		// limit to 5 attempts in case something goes wrong (a bug in the
		// randomiser) or we somehow win the lottery and keep generating valid
		// DOIs
		if ntry == 5 {
			resData.Success = false
			resData.Level = "warning"
			resData.Message = template.HTML(msgSubmitError)
			renderResult(w, &resData)
			return

		}
		doi = conf.DOIBase + randAlnum(6)
	}

	// NOTE: Delete?
	_, err = conf.GIN.Session.RequestAccount(requser.Username)
	if err != nil {
		// Can happen if the DOI service isn't logged in to GIN
		log.Printf("Failed to get user data: %s", err.Error())
		log.Printf("Request data: %+v", reqdata)
		errors = append(errors, fmt.Sprintf("Failed to get user data: %s", err.Error()))
		resData.Success = true
		resData.Level = "warning"
		resData.Message = template.HTML(msgSubmitError)
		return
	}

	infoyml, err := readFileAtURL(dataciteURL(regJob.Metadata.SourceRepository, conf))
	if err != nil {
		// Can happen if the datacite.yml file or the repository is removed (or
		// made private) between preparing the request and submitting it
		log.Printf("Failed to fetch datacite.yml: %s", err.Error())
		log.Printf("Request data: %+v", reqdata)
		errors = append(errors, fmt.Sprintf("Failed to fetch datacite.yml: %s", err.Error()))
		resData.Success = true
		resData.Level = "warning"
		resData.Message = template.HTML(msgSubmitError)
		return
	}
	yamlInfo, err := readRepoYAML(infoyml)
	if err != nil {
		// Can happen if the datacite.yml file is modified (and made invalid)
		// between preparing the request and submitting it
		log.Printf("Failed to parse datacite.yml: %s", err.Error())
		log.Printf("Request data: %+v", reqdata)
		errors = append(errors, fmt.Sprintf("Failed to parse datacite.yml: %s", err.Error()))
		resData.Success = true
		resData.Level = "warning"
		resData.Message = template.HTML(msgSubmitError)
		return
	}

	regJob.Metadata.YAMLData = yamlInfo
	regJob.Metadata.DataCite = libgin.NewDataCiteFromYAML(yamlInfo)
	regJob.Metadata.Identifier.ID = doi
	regJob.Metadata.Identifier.Type = "DOI"

	log.Printf("Submitting job")

	// Add job to queue
	jobQueue <- regJob

	// Render success (deferred)
	log.Printf("Render success")
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
	encReqData := r.Form.Get("regrequest")

	log.Printf("Got request: %s", encReqData)

	regRequest := &RegistrationRequest{}
	reqdata, err := decryptRequestData(encReqData, conf.Key)
	if err != nil {
		log.Printf("Invalid request: %s", err.Error())
		w.WriteHeader(http.StatusBadRequest)
		regRequest.Message = template.HTML(msgInvalidRequest)
		regRequest.Metadata = &libgin.RepositoryMetadata{}
		tmpl, err := template.New("requestpage").Parse(gdtmpl.RequestFailurePage)
		if err != nil {
			log.Printf("Failed to parse requestpage template: %s", err.Error())
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		tmpl.Execute(w, regRequest)
		return
	}

	regRequest.DOIRequestData = reqdata
	regRequest.EncryptedRequestData = encReqData // Forward it through the hidden form in the template
	regRequest.Metadata = &libgin.RepositoryMetadata{}

	infoyml, err := readFileAtURL(dataciteURL(regRequest.Repository, conf))
	if err != nil {
		// Can happen if the datacite.yml file is removed and the user clicks DOIfy on a stale page
		log.Printf("Failed to fetch datacite.yml: %s", err.Error())
		log.Printf("Request data: %+v", regRequest)
		regRequest.ErrorMessages = []string{fmt.Sprintf("Failed to fetch datacite.yml: %s", err.Error())}
		regRequest.Message = template.HTML(msgInvalidDOI + " <p><i>No datacite.yml file found in repository</i>")
		tmpl, err := template.New("requestpage").Parse(gdtmpl.RequestFailurePage)
		if err != nil {
			log.Printf("Failed to parse requestpage template: %s", err.Error())
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		tmpl.Execute(w, regRequest)
		return
	}
	doiInfo, err := readRepoYAML(infoyml)
	if err != nil {
		log.Print("DOI file invalid")
		regRequest.Message = template.HTML(msgInvalidDOI + " <p><i>" + err.Error() + "</i>")
		tmpl, err := template.New("requestpage").Parse(gdtmpl.RequestFailurePage)
		if err != nil {
			log.Printf("Failed to parse requestpage template: %s", err.Error())
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		tmpl.Execute(w, regRequest)
		return
	}

	// All good: Render request page
	tmpl, err := template.New("doiInfo").Funcs(tmplfuncs).Parse(gdtmpl.DOIInfo)
	if err != nil {
		log.Printf("Failed to parse DOI info template: %s", err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	tmpl, err = tmpl.New("requestpage").Parse(gdtmpl.RequestPage)
	if err != nil {
		log.Printf("Failed to parse requestpage template: %s", err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		// TODO: Notify via email
		return
	}

	j, _ := json.MarshalIndent(doiInfo, "", "  ")
	log.Printf("Received DOI information: %s", string(j))

	regRequest.Metadata.YAMLData = doiInfo
	regRequest.Metadata.DataCite = libgin.NewDataCiteFromYAML(doiInfo)
	regRequest.Metadata.SourceRepository = regRequest.DOIRequestData.Repository
	regRequest.Metadata.ForkRepository = "" // not forked yet

	err = tmpl.Execute(w, regRequest)
	if err != nil {
		log.Printf("Error rendering template: %s", err.Error())
	}
}

// decryptRequestData decrypts the submitted data into a map.  Returns with
// error if the decryption fails, the encrypted data is not a valid JSON
// object, or if any of the expected keys (username, realname, repository,
// email) are not present.
func decryptRequestData(regrequest string, key string) (*libgin.DOIRequestData, error) {
	plaintext, err := libgin.DecryptURLString([]byte(key), regrequest)
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

func web(cmd *cobra.Command, args []string) {
	log.Printf("Starting up %s", cmd.Version)

	config, err := loadconfig()
	if err != nil {
		log.Fatalf("Startup failed: %v", err)
	}

	// Pretty print configuration for debugging, but hide sensitive stuff
	cc := *config
	cc.Key = "[HIDDEN]"
	cc.GIN.Password = "[HIDDEN]"
	j, _ := json.MarshalIndent(cc, "", "  ")
	log.Print(string(j))

	log.Printf("Logging in to GIN (%s) as %s", config.GIN.Session.WebAddress(), config.GIN.Username)
	err = config.GIN.Session.Login(config.GIN.Username, config.GIN.Password, "gin-doi")
	if err != nil {
		log.Fatal(err)
	}

	defer config.GIN.Session.Logout()

	jobQueue := make(chan *RegistrationJob, config.MaxQueue)
	dispatcher := newDispatcher(jobQueue, config.MaxWorkers)
	dispatcher.run(newWorker)

	// Start the HTTP handlers.

	// Root redirects to storage URL (DOI listing page)
	http.Handle("/", http.RedirectHandler(config.Storage.StoreURL, http.StatusMovedPermanently))

	// register renders the info page with the registration button
	http.HandleFunc("/register", func(w http.ResponseWriter, r *http.Request) {
		log.Printf("Got request: %s", r.URL.String())
		renderRequestPage(w, r, config)
	})

	// submit starts the registration job
	http.HandleFunc("/submit", func(w http.ResponseWriter, r *http.Request) {
		startDOIRegistration(w, r, jobQueue, config)
	})

	// assets fetches static assets using a custom FileSystem
	assetserver := http.FileServer(newAssetFS("/assets"))
	http.Handle("/assets/", http.StripPrefix("/assets/", assetserver))

	fmt.Printf("Listening for connections on port %d\n", config.Port)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", config.Port), nil))
}
