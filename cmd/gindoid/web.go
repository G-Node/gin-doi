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

// injectDynamicGINURL overwrites the 'GINServerURL' default template function
// to provide a dynamic GIN server URL.
func injectDynamicGINURL(tmpl *template.Template, ginurl string) *template.Template {
	if ginurl != "" {
		var injectedFunc = template.FuncMap{
			"GINServerURL": func() string {
				return ginurl
			},
		}
		// Clone template to avoid race condition when setting injected FuncMap
		tmpl = template.Must(tmpl.Clone()).Funcs(injectedFunc)
	}
	return tmpl
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
		// Overwrite default GIN server URL with config GIN server URL
		tmpl = injectDynamicGINURL(tmpl, GetGINURL(conf))
		err = tmpl.Execute(w, regRequest)
		if err != nil {
			log.Printf("Failed to execute RequestFailurePage template: %q", err.Error())
		}
		return
	}

	regRequest.DOIRequestData = reqdata
	regRequest.EncryptedRequestData = encReqData // Forward it through the hidden form in the template
	regRequest.Metadata = &libgin.RepositoryMetadata{}

	repoMetadata, err := readAndValidate(conf, regRequest.Repository)
	if err != nil {
		regRequest.ErrorMessages = []string{err.Error()}
		regRequest.Message = template.HTML(err.Error())
		tmpl, err := prepareTemplates("RequestFailurePage")
		if err != nil {
			log.Printf("Failed to parse RequestFailurePage template: %s", err.Error())
			log.Printf("Request data: %+v", regRequest)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		// Overwrite default GIN server URL with config GIN server URL
		tmpl = injectDynamicGINURL(tmpl, GetGINURL(conf))
		err = tmpl.Execute(w, regRequest)
		if err != nil {
			log.Printf("Failed to execute RequestFailurePage template: %q", err.Error())
		}
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
	regRequest.Metadata.ForkRepository = regRequest.DOIRequestData.Repository // Make the button link to repo for preview

	// Overwrite default GIN server URL with config GIN server URL
	tmpl = injectDynamicGINURL(tmpl, GetGINURL(conf))
	err = tmpl.Execute(w, regRequest)
	if err != nil {
		log.Printf("Error rendering template: %s", err.Error())
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
		renderResult(w, &resData, conf)
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

	// Get the latest repository commit hash before the whole procedure to
	// ensure long clone and zip processes do not obscure a commit change
	// after the fact;
	// on error, an empty string is returned as commithash
	commithash, err := getRepoCommit(conf.GIN.Session, reqdata.Repository)
	if err != nil {
		log.Printf("%s", err.Error())
	}

	// exiting beyond this point should trigger an email notification
	defer func() {
		// This is the first notification, so include the entire info
		err := notifyAdmin(regJob, errors, nil, true, commithash)
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
		renderResult(w, &resData, conf)
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

	repoMetadata, err := readAndValidate(conf, regJob.Metadata.SourceRepository)
	if err != nil {
		errors = append(errors, err.Error())
		resData.Success = false
		resData.Level = "error"
		resData.Message = template.HTML(err.Error())
		return
	}

	// Permit DOI references without prefix without updating the libgin library;
	// this code snippet can be removed if the corresponding libgin function
	// (libgin.DataCite.AddReference) is updated and a new library version is released.
	// This handling takes care of references containing only DOI URLs as well
	// as improperly formatted DOI references entries that end with a proper DOI URL
	// e.g: 'id: "doi:  https://doi.org/some-doi"'.
	doiSplit := "https://doi.org/"
	for idx, ref := range repoMetadata.References {
		if strings.Contains(ref.ID, doiSplit) {
			log.Printf("Updating DOI reference ID %q", ref.ID)
			doiID := strings.Split(ref.ID, doiSplit)
			repoMetadata.References[idx].ID = fmt.Sprintf("doi:%s", doiID[1])
		}
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

	// Send user email notification
	if err := notifyUser(regJob); err != nil {
		// Inform admins that user email failed
		errors = append(errors, fmt.Sprintf("Failed to send user notification email: %s", err.Error()))
	}
}

// renderResult renders the results of a registration request using the
// 'RequestResult' template. If it fails to parse the template, it renders
// the Message from the result data in plain HTML.
func renderResult(w http.ResponseWriter, resData *reqResultData, conf *Configuration) {
	tmpl, err := prepareTemplates("RequestResult")
	if err != nil {
		log.Printf("Failed to parse requestresult template: %s", err.Error())
		log.Printf("Request data: %+v", resData)
		// failed to render result template; just show the message wrapped in html tags
		_, err = w.Write([]byte("<html>" + resData.Message + "</html>"))
		if err != nil {
			log.Printf("Failed to write fallback page: %s", err.Error())
		}
		return
	}
	// Overwrite default GIN server URL with config GIN server URL
	tmpl = injectDynamicGINURL(tmpl, GetGINURL(conf))
	err = tmpl.Execute(w, &resData)
	if err != nil {
		log.Printf("Error rendering RequestResult template: %v", err.Error())
	}
}
