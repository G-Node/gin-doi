package main

import (
	"fmt"
	"html/template"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/G-Node/gin-cli/git"
	"github.com/G-Node/libgin/libgin"
	"github.com/G-Node/libgin/libgin/archive"
	humanize "github.com/dustin/go-humanize"
	yaml "gopkg.in/yaml.v2"
)

const (
	tmpdir      = "tmp"
	doixmlfname = "datacite.xml"
)

// createRegisteredDataset starts the process of registering a dataset. It's
// the top level function for the dataset registration and calls all other
// individual functions.
func createRegisteredDataset(job *RegistrationJob) error {
	conf := job.Config
	repopath := job.Metadata.SourceRepository
	jobname := job.Metadata.Identifier.ID

	prepDir(job)

	targetpath := filepath.Join(conf.Storage.TargetDirectory, jobname)
	preperrors := make([]string, 0, 5)

	ginurl, err := url.Parse(GetGINURL(conf))
	if err != nil {
		preperrors = append(preperrors, fmt.Sprintf("Bad GIN URL configured: %s", err.Error()))
	}

	ginurl.Path = job.Metadata.SourceRepository
	repoURL := ginurl.String()
	ginurl.Path = job.Metadata.ForkRepository
	forkURL := ginurl.String()

	zipfname, zipsize, err := cloneAndZip(repopath, jobname, targetpath, conf)
	var archiveURL string
	if err != nil {
		// failed to clone and zip
		// save the error for reporting and continue with the XML prep
		preperrors = append(preperrors, err.Error())
	} else if storeURL, err := url.Parse(conf.Storage.StoreURL); err == nil {
		storeURL.Path = path.Join(job.Metadata.Identifier.ID, zipfname)
		archiveURL = storeURL.String()
		job.Metadata.Sizes = &[]string{humanize.IBytes(uint64(zipsize))}
	} else {
		preperrors = append(preperrors, fmt.Sprintf("zip file created, but failed to parse StoreURL: %s", err.Error()))
	}
	job.Metadata.AddURLs(repoURL, forkURL, archiveURL)

	createLandingPage(job.Metadata, filepath.Join(conf.Storage.TargetDirectory, job.Metadata.Identifier.ID, "index.html"))

	fp, err := os.Create(filepath.Join(targetpath, "doi.xml"))
	if err != nil {
		log.Print("Could not create the metadata template")
		// XML Creation failed; return with error
		preperrors = append(preperrors, fmt.Sprintf("Failed to create the XML metadata template: %s", err))
		notifyAdmin(job, preperrors, nil)
		return err
	}
	defer fp.Close()

	data, err := job.Metadata.DataCite.Marshal()
	if err != nil {
		log.Print("Could not render the metadata file")
		preperrors = append(preperrors, fmt.Sprintf("Failed to render the XML metadata: %s", err))
		notifyAdmin(job, preperrors, nil)
		return err
	}
	_, err = fp.Write([]byte(data))
	if err != nil {
		log.Print("Could not write to the metadata file")
		preperrors = append(preperrors, fmt.Sprintf("Failed to write the metadata XML file: %s", err))
	}

	warnings := collectWarnings(job)

	if len(preperrors)+len(warnings) > 0 {
		// Resend email with errors if any occurred
		notifyAdmin(job, preperrors, warnings)
	}
	return err
}

// cloneAndZip clones the source repository into a temporary directory under
// targetpath, zips the contents, and returns the archive filename and its size
// in bytes.
func cloneAndZip(repopath string, jobname string, targetpath string, conf *Configuration) (string, int64, error) {
	// Clone under targetpath (will create subdirectory with repository name)
	if err := os.MkdirAll(targetpath, 0777); err != nil {
		errmsg := fmt.Sprintf("Failed to create temporary clone directory: %s", tmpdir)
		log.Print(errmsg)
		return "", -1, fmt.Errorf(errmsg)
	}

	// Clone
	if err := cloneRepo(repopath, targetpath, conf); err != nil {
		log.Print("Repository cloning failed")
		return "", -1, fmt.Errorf("Failed to clone repository '%s': %v", repopath, err)
	}

	// Uninit the annex and delete .git directory
	repoparts := strings.SplitN(repopath, "/", 2)
	reponame := strings.ToLower(repoparts[1]) // clone directory is always lowercase
	repodir := filepath.Join(targetpath, reponame)
	if err := derepoCloneDir(repodir); err != nil {
		log.Print("Repository cleanup (uninit & derepo) failed")
		return "", -1, fmt.Errorf("Failed to uninit and cleanup repository '%s': %v", repopath, err)
	}

	// Zip
	log.Printf("Preparing zip file for %s", jobname)
	// use DOI with / replacement for zip filename
	zipbasename := strings.ReplaceAll(jobname, "/", "_") + ".zip"
	zipfilename := filepath.Join(targetpath, zipbasename)
	zipsize, err := zip(repodir, zipfilename)
	if err != nil {
		log.Print("Could not zip the data")
		return "", -1, fmt.Errorf("Failed to create the zip file: %v", err)
	}
	log.Printf("Archive size: %d", zipsize)
	return zipbasename, zipsize, nil
}

// zip a source directory into a file with the given filename.
func zip(source, zipfilename string) (int64, error) {
	fn := fmt.Sprintf("zip(%s, %s)", source, zipfilename) // keep original args for errmsg
	source, err := filepath.Abs(source)
	if err != nil {
		log.Printf("%s: Failed to get abs path for source directory in function '%s': %v", lpStorage, fn, err)
		return -1, err
	}

	zipfilename, err = filepath.Abs(zipfilename)
	if err != nil {
		log.Printf("%s: Failed to get abs path for target zip file in function '%s': %v", lpStorage, fn, err)
		return -1, err
	}

	// Create zip file IO writer for MakeZip function
	zipfp, err := os.Create(zipfilename)
	if err != nil {
		log.Printf("%s: Failed to create zip file for writing in function '%s': %v", lpStorage, fn, err)
		return -1, err
	}
	defer zipfp.Close()
	// Change into clone directory to make the paths in the zip archive repo
	// root-relative.
	origdir, err := os.Getwd()
	if err != nil {
		log.Printf("%s: Failed to get working directory in function '%s': %v", lpStorage, fn, err)
		return -1, err
	}
	defer os.Chdir(origdir)
	if err := os.Chdir(source); err != nil {
		log.Printf("%s: Failed to change to source directory to make zip file in function '%s': %v", lpStorage, fn, err)
		return -1, err
	}

	if err := archive.MakeZip(zipfp, "."); err != nil {
		log.Printf("%s: Failed to create zip file in function '%s': %v", lpStorage, fn, err)
		return -1, err
	}

	stat, _ := zipfp.Stat()
	return stat.Size(), nil
}

// createLandingPage renders and writes a registered dataset landing page based
// on the LandingPage template.
func createLandingPage(metadata *libgin.RepositoryMetadata, targetfile string) error {
	tmpl, err := prepareTemplates("DOIInfo", "LandingPage")
	if err != nil {
		return err
	}
	fp, err := os.Create(targetfile)
	if err != nil {
		log.Printf("Could not create the landing page file: %s", err.Error())
		return err
	}
	defer fp.Close()
	if err := tmpl.Execute(fp, metadata); err != nil {
		log.Printf("Error rendering the landing page: %s", err.Error())
		return err
	}
	return nil
}

// prepDir creates the directory where the dataset will be cloned and archived.
func prepDir(job *RegistrationJob) error {
	conf := job.Config
	metadata := job.Metadata
	storagedir := conf.Storage.TargetDirectory
	doi := metadata.Identifier.ID
	err := os.MkdirAll(filepath.Join(storagedir, doi), os.ModePerm)
	if err != nil {
		log.Print("Could not create the target directory")
		return err
	}
	// Deny access per default
	file, err := os.Create(filepath.Join(storagedir, doi, ".htaccess"))
	if err != nil {
		log.Print("Could not create .htaccess")
		return err
	}
	defer file.Close()
	// todo check
	_, err = file.Write([]byte("deny from all"))
	if err != nil {
		log.Print("Could not write to .htaccess")
		return err
	}
	return nil
}

// derepoCloneDir de-initialises the annex in a repository and deletes the .git
// directory.
func derepoCloneDir(directory string) error {
	directory, err := filepath.Abs(directory)
	if err != nil {
		log.Printf("%s: Failed to get abs path for repo directory while cleaning up '%s'. Was our working directory removed?", lpStorage, directory)
		return err
	}
	// NOTE: Most of the functionality in this method will be moved to libgin
	// since GOGS has similar functions
	// Change into directory to cleanup and defer changing back
	origdir, err := os.Getwd()
	if err != nil {
		log.Printf("%s: Failed to get abs path for working directory while cleaning up directory '%s'. Was our working directory removed?", lpStorage, directory)
		return err
	}
	defer os.Chdir(origdir)
	if err := os.Chdir(directory); err != nil {
		log.Printf("%s: Failed to change working directory to '%s': %v", lpStorage, directory, err)
		return err
	}

	// Uninit annex
	cmd := git.AnnexCommand("uninit")
	// git annex uninit always returns with an error (-_-) so we ignore the
	// error and check if annex info complains instead
	cmd.Run()

	_, err = git.AnnexInfo()
	if err != nil {
		log.Printf("%s: Failed to uninit annex in cloned repository '%s': %v", lpStorage, directory, err)
	}

	gitdir, err := filepath.Abs(filepath.Join(directory, ".git"))
	if err != nil {
		log.Printf("%s: Failed to get abs path for git directory while cleaning up directory '%s'. Was our working directory removed?", lpStorage, directory)
		return err
	}
	// Set write permissions on everything under gitdir
	var mode os.FileMode
	walker := func(path string, info os.FileInfo, err error) error {
		// walker sets the permission for any file found to 0660 and directories to
		// 770, to allow deletion
		if info == nil {
			return nil
		}

		mode = 0660
		if info.IsDir() {
			mode = 0770
		}

		if err := os.Chmod(path, mode); err != nil {
			log.Printf("failed to change permissions on '%s': %v", path, err)
		}
		return nil
	}
	if err := filepath.Walk(gitdir, walker); err != nil {
		log.Printf("%s: Failed to set write permissions for directories and files under gitdir '%s': %v", lpStorage, gitdir, err)
		return err
	}

	// Delete .git directory
	if err := os.RemoveAll(gitdir); err != nil {
		log.Printf("%s: Failed to remove git directory '%s': %v", lpStorage, gitdir, err)
		return err
	}

	return nil
}

// cloneRepo clones a git repository (with git-annex) specified by URI to the
// destination directory.
func cloneRepo(URI string, destdir string, conf *Configuration) error {
	// NOTE: cloneRepo changes the working directory to the cloned repository
	// See: https://github.com/G-Node/gin-cli/issues/225
	// This will need to change when that issue is fixed
	origdir, err := os.Getwd()
	if err != nil {
		log.Printf("%s: Failed to get working directory when cloning repository. Was our working directory removed?", lpStorage)
		return err
	}
	defer os.Chdir(origdir)
	err = os.Chdir(destdir)
	if err != nil {
		return err
	}
	log.Printf("Cloning %s", URI)

	clonechan := make(chan git.RepoFileStatus)
	go conf.GIN.Session.CloneRepo(strings.ToLower(URI), clonechan)
	for stat := range clonechan {
		log.Print(stat)
		if stat.Err != nil {
			log.Printf("Repository cloning failed: %s", stat.Err)
			return stat.Err
		}
	}

	downloadchan := make(chan git.RepoFileStatus)
	go conf.GIN.Session.GetContent(nil, downloadchan)
	for stat := range downloadchan {
		log.Print(stat)
		if stat.Err != nil {
			log.Printf("Repository cloning failed during annex get: %s", stat.Err)
			return stat.Err
		}
	}
	return nil
}

// repoFileURL returns the full URL to a file on the master branch of a
// repository.
func repoFileURL(conf *Configuration, repopath string, filename string) string {
	u, err := url.Parse(GetGINURL(conf))
	if err != nil {
		// not configured properly; return nothing
		return ""
	}
	fetchRepoPath := fmt.Sprintf("%s/raw/master/%s", repopath, filename)
	u.Path = fetchRepoPath
	return u.String()
}

// readFileAtURL returns the contents of a file at a given URL.
func readFileAtURL(url string) ([]byte, error) {
	client := &http.Client{}
	log.Printf("Fetching file at %q", url)
	req, _ := http.NewRequest(http.MethodGet, url, nil)
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("Request failed: %s", err.Error())
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Request returned non-OK status: %s", resp.Status)
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Printf("Could not read file contents: %s", err.Error())
		return nil, err
	}
	return body, nil
}

// readRepoYAML parses the DOI registration info and returns a filled DOIRegInfo struct.
func readRepoYAML(infoyml []byte) (*libgin.RepositoryYAML, error) {
	yamlInfo := &libgin.RepositoryYAML{}
	err := yaml.Unmarshal(infoyml, yamlInfo)
	if err != nil {
		return nil, fmt.Errorf("error while reading DOI info: %s", err.Error())
	}
	if missing := checkMissingValues(yamlInfo); len(missing) > 0 {
		log.Print("DOI file is missing entries")
		return nil, fmt.Errorf(strings.Join(missing, " "))
	}
	return yamlInfo, nil
}

// RegistrationRequest holds the encrypted and decrypted data of a registration
// request, as well as the unmarshalled data of the target repository's
// datacite.yml metadata.  It's used to render the preparation page (request
// page) for the user to review the metadata before finalising the request.
type RegistrationRequest struct {
	// Encrypted request data from GIN.
	EncryptedRequestData string
	// Decrypted and unmarshalled request data.
	*libgin.DOIRequestData
	// Used to display error or warning messages to the user through the templates.
	Message template.HTML
	// Metadata for the repository being registered
	Metadata *libgin.RepositoryMetadata
	// Errors during the registration process that get sent in the body of the
	// email to the administrators.
	ErrorMessages []string
}

func (d *RegistrationRequest) GetDOIURI() string {
	var re = regexp.MustCompile(`(.+)\/`)
	return string(re.ReplaceAll([]byte(d.Repository), []byte("doi/")))
}

func (d *RegistrationRequest) AsHTML() template.HTML {
	return template.HTML(d.Message)
}

// readAndValidate loads the datacite.yml file at the given URL, validates it
// and returns the RepositoryYAML struct or an error message if the retrieval,
// parsing, or validation fails.  The message is appropriate for display to the
// user.
func readAndValidate(conf *Configuration, repository string) (*libgin.RepositoryYAML, error) {
	dataciteText, err := readFileAtURL(repoFileURL(conf, repository, "datacite.yml"))
	if err != nil {
		// Can happen if the datacite.yml file is removed and the user clicks the register button on a stale page
		err := fmt.Errorf("%s <p><i>No datacite.yml file found in repository</i></p>", msgInvalidDOI)
		return nil, err
	}

	repoMetadata, err := readRepoYAML(dataciteText)
	if err != nil {
		log.Print("DOI file invalid")
		err := fmt.Errorf("%s<p><i>%s</i></p>", msgInvalidDOI, err.Error())
		return nil, err
	}

	licenseText, err := readFileAtURL(repoFileURL(conf, repository, "LICENSE"))
	if err != nil {
		log.Printf("Failed to fetch LICENSE: %s", err.Error())
		return nil, fmt.Errorf(msgNoLicenseFile)
	}

	expectedTextURL := repoFileURL(conf, "G-Node/Info", fmt.Sprintf("licenses/%s", repoMetadata.License.Name))
	if !checkLicenseMatch(expectedTextURL, string(licenseText)) {
		// License file doesn't match specified license
		errmsg := fmt.Sprintf("License file does not match specified license: %q", repoMetadata.License.Name)
		log.Print(errmsg)
		return nil, fmt.Errorf(msgLicenseMismatch)
	}

	if msgs := validateDataCiteValues(repoMetadata); len(msgs) > 0 {
		return nil, fmt.Errorf(strings.Join(msgs, "\n"))
	}

	return repoMetadata, nil
}
