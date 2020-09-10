package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"path"
	"strings"

	"github.com/G-Node/libgin/libgin"
	"github.com/spf13/cobra"
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
	tmpl, err := prepareTemplates("RequestResult")
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

	// Fully initialise nested regJob in case something goes wrong
	// Uninitialised child ptrs might panic during error reporting
	regJob := &RegistrationJob{
		Metadata: new(libgin.RepositoryMetadata),
		Config:   conf,
	}
	regJob.Metadata.DataCite = new(libgin.DataCite)
	resData := reqResultData{}

	encryptedRequestData := r.PostFormValue("reqdata")
	reqdata, err := decryptRequestData(encryptedRequestData, conf.Key)
	if err != nil {
		log.Printf("Invalid request: %s", err.Error())
		resData.Message = template.HTML(msgInvalidRequest)
		// ignore the error, no email to send
		renderResult(w, &resData)
		return
	}
	resData.Repository = reqdata.Repository

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
		regJob.Metadata.ForkRepository = path.Join("doi", repoParts[1])
	}
	// otherwise, unexpected repository name, so don't set ForkRepository and
	// the cloner will notify

	// exiting beyond this point should trigger an email notification
	defer func() {
		err := notifyAdmin(regJob, errors, nil)
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
	maxtry := 5
	for ntry := 0; doi == "" || libgin.IsRegisteredDOI(doi); ntry++ {
		// limit to 5 attempts in case something goes wrong (a bug in the
		// randomiser) or we somehow win the lottery and keep generating valid
		// DOIs
		if ntry == maxtry {
			errors = append(errors, fmt.Sprintf("Couldn't find a new DOI after %d tries (or the PRNG is broken)", maxtry))
			resData.Success = false
			resData.Level = "warning"
			resData.Message = template.HTML(msgSubmitError)
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

	dataciteText, err := readFileAtURL(repoFileURL(conf, regJob.Metadata.SourceRepository, "datacite.yml"))
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
	repoMetadata, err := readRepoYAML(dataciteText)
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

	licenseText, err := readFileAtURL(repoFileURL(conf, regJob.Metadata.SourceRepository, "LICENSE"))
	if err != nil {
		// No license file
		log.Printf("Failed to fetch LICENSE: %s", err.Error())
		log.Printf("Request data: %+v", reqdata)
		errors = append(errors, fmt.Sprintf("Failed to fetch LICENSE: %s", err.Error()))
		resData.Success = false
		resData.Level = "warning"
		resData.Message = template.HTML(msgNoLicenseFile)
		return
	}

	expectedTextURL := repoFileURL(conf, "G-Node/Info", fmt.Sprintf("licenses/%s", repoMetadata.License.Name))
	if !checkLicenseMatch(expectedTextURL, string(licenseText)) {
		// License file doesn't match specified license
		errmsg := fmt.Sprintf("License file does not match specified license: %q", repoMetadata.License.Name)
		log.Print(errmsg)
		log.Printf("Request data: %+v", reqdata)
		errors = append(errors, errmsg)
		resData.Success = false
		resData.Level = "warning"
		resData.Message = template.HTML(msgLicenseMismatch)
		return
	}

	regJob.Metadata.YAMLData = repoMetadata
	regJob.Metadata.DataCite = libgin.NewDataCiteFromYAML(repoMetadata)
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
		regRequest.Metadata = new(libgin.RepositoryMetadata)
		tmpl, err := prepareTemplates("RequestFailurePage")
		if err != nil {
			log.Printf("Failed to parse RequestFailurePage template: %s", err.Error())
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		tmpl.Execute(w, regRequest)
		return
	}

	regRequest.DOIRequestData = reqdata
	regRequest.EncryptedRequestData = encReqData // Forward it through the hidden form in the template
	regRequest.Metadata = &libgin.RepositoryMetadata{}

	dataciteText, err := readFileAtURL(repoFileURL(conf, regRequest.Repository, "datacite.yml"))
	if err != nil {
		// Can happen if the datacite.yml file is removed and the user clicks the register button on a stale page
		log.Printf("Failed to fetch datacite.yml: %s", err.Error())
		log.Printf("Request data: %+v", regRequest)
		regRequest.ErrorMessages = []string{fmt.Sprintf("Failed to fetch datacite.yml: %s", err.Error())}
		regRequest.Message = template.HTML(msgInvalidDOI + " <p><i>No datacite.yml file found in repository</i>")
		tmpl, err := prepareTemplates("RequestFailurePage")
		if err != nil {
			log.Printf("Failed to parse requestpage template: %s", err.Error())
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		tmpl.Execute(w, regRequest)
		return
	}

	repoMetadata, err := readRepoYAML(dataciteText)
	if err != nil {
		log.Print("DOI file invalid")
		regRequest.Message = template.HTML(msgInvalidDOI + " <p><i>" + err.Error() + "</i>")
		tmpl, err := prepareTemplates("RequestFailurePage")
		if err != nil {
			log.Printf("Failed to parse requestpage template: %s", err.Error())
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		tmpl.Execute(w, regRequest)
		return
	}

	licenseText, err := readFileAtURL(repoFileURL(conf, regRequest.Repository, "LICENSE"))
	if err != nil {
		log.Printf("Failed to fetch LICENSE: %s", err.Error())
		log.Printf("Request data: %+v", regRequest)
		regRequest.ErrorMessages = []string{fmt.Sprintf("Failed to fetch LICENSE: %s", err.Error())}
		regRequest.Message = template.HTML(msgNoLicenseFile)
		tmpl, err := prepareTemplates("RequestFailurePage")
		if err != nil {
			log.Printf("Failed to parse requestpage template: %s", err.Error())
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		tmpl.Execute(w, regRequest)
		return
	}

	expectedTextURL := repoFileURL(conf, "G-Node/Info", fmt.Sprintf("licenses/%s", repoMetadata.License.Name))
	if !checkLicenseMatch(expectedTextURL, string(licenseText)) {
		// License file doesn't match specified license
		errmsg := fmt.Sprintf("License file does not match specified license: %q", repoMetadata.License.Name)
		log.Print(errmsg)
		log.Printf("Request data: %+v", reqdata)
		regRequest.Message = template.HTML(msgLicenseMismatch)
		tmpl, err := prepareTemplates("RequestFailurePage")
		if err != nil {
			log.Printf("Failed to parse requestpage template: %s", err.Error())
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		tmpl.Execute(w, regRequest)
		return
	}

	// All good: Render request page
	tmpl, err := prepareTemplates("DOIInfo", "RequestPage")
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		// TODO: Notify via email
		return
	}

	j, _ := json.MarshalIndent(repoMetadata, "", "  ")
	log.Printf("Received DOI information: %s", string(j))

	regRequest.Metadata.YAMLData = repoMetadata
	regRequest.Metadata.DataCite = libgin.NewDataCiteFromYAML(repoMetadata)
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
