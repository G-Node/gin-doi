package main

import (
	"archive/zip"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/G-Node/gin-cli/ginclient"
	"github.com/G-Node/gin-cli/git"
	"github.com/G-Node/libgin/libgin"
	humanize "github.com/dustin/go-humanize"
	"github.com/gogs/go-gogs-client"
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

	preppath := filepath.Join(conf.Storage.PreparationDirectory, jobname)
	zipfname, zipsize, err := cloneAndZip(repopath, jobname, preppath, targetpath, conf)
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

	// Check if there are older versions of the same dataset
	if oldID := getPreviousDOI(job); oldID != "" {
		relatedIdentifier := libgin.RelatedIdentifier{Identifier: oldID, Type: "DOI", RelationType: "IsNewVersionOf"}
		job.Metadata.RelatedIdentifiers = append(job.Metadata.RelatedIdentifiers, relatedIdentifier)
	}

	dynurl := GetGINURL(conf)
	createLandingPage(job.Metadata, filepath.Join(conf.Storage.TargetDirectory, job.Metadata.Identifier.ID, "index.html"), dynurl)

	fp, err := os.Create(filepath.Join(targetpath, "doi.xml"))
	if err != nil {
		log.Print("Could not create the metadata template")
		// XML Creation failed; return with error
		preperrors = append(preperrors, fmt.Sprintf("Failed to create the XML metadata template: %s", err))
		notifyAdmin(job, preperrors, nil, false)
		return err
	}
	defer fp.Close()

	data, err := job.Metadata.DataCite.Marshal()
	if err != nil {
		log.Print("Could not render the metadata file")
		preperrors = append(preperrors, fmt.Sprintf("Failed to render the XML metadata: %s", err))
		notifyAdmin(job, preperrors, nil, false)
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
		notifyAdmin(job, preperrors, warnings, false)
	}
	return err
}

// cloneAndZip clones the source repository into a temporary directory under
// preppath, zips the contents at the targetpath, and returns the archive filename
// and its size in bytes.
func cloneAndZip(repopath string, jobname string, preppath string, targetpath string, conf *Configuration) (string, int64, error) {
	log.Print("Start clone and zip")
	// Clone at preppath (will create subdirectories '[doi-org-id]/[doi-jobname]/[reponame]')
	if err := os.MkdirAll(preppath, 0777); err != nil {
		errmsg := fmt.Sprintf("Failed to create temporary clone directory: %s", tmpdir)
		log.Print(errmsg)
		return "", -1, fmt.Errorf(errmsg)
	}

	// Clone repository at the preparation path
	if err := cloneRepo(repopath, preppath, conf); err != nil {
		log.Print("Repository cloning failed")
		return "", -1, fmt.Errorf("Failed to clone repository '%s': %v", repopath, err)
	}

	// Zip repository content to the target path
	repoparts := strings.SplitN(repopath, "/", 2)
	reponame := strings.ToLower(repoparts[1]) // clone directory is always lowercase
	repodir := filepath.Join(preppath, reponame)

	log.Printf("Preparing zip file for %s", jobname)
	// use DOI with / replacement for zip filename
	zipbasename := strings.ReplaceAll(jobname, "/", "_") + ".zip"
	zipfilename := filepath.Join(targetpath, zipbasename)
	// exclude the git folder from the zip file
	exclude := []string{".git"}
	zipsize, err := runzip(repodir, zipfilename, exclude)
	if err != nil {
		log.Print("Could not zip the data")
		return "", -1, fmt.Errorf("Failed to create the zip file: %v", err)
	}
	log.Printf("Archive size: %d", zipsize)
	return zipbasename, zipsize, nil
}

// runzip zips a source directory into a file with the given filename.  Any directories
// or files handed over via the exclude parameter will not be zipped.
func runzip(source, zipfilename string, exclude []string) (int64, error) {
	fn := fmt.Sprintf("runzip(%s, %s)", source, zipfilename) // keep original args for errmsg
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

	if err := MakeZip(zipfp, exclude, "."); err != nil {
		log.Printf("%s: Failed to create zip file in function '%s': %v", lpStorage, fn, err)
		return -1, err
	}

	stat, _ := zipfp.Stat()
	return stat.Size(), nil
}

// createLandingPage renders and writes a registered dataset landing page based
// on the LandingPage template.
func createLandingPage(metadata *libgin.RepositoryMetadata, targetfile string, ginurl string) error {
	tmpl, err := prepareTemplates("DOIInfo", "LandingPage")
	if err != nil {
		return err
	}
	// Overwrite default GIN server URL with config GIN server URL
	tmpl = injectDynamicGINURL(tmpl, ginurl)

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

// prepDir creates the directories where the dataset will be cloned and archived.
func prepDir(job *RegistrationJob) error {
	conf := job.Config
	metadata := job.Metadata
	prepdir := conf.Storage.PreparationDirectory
	storagedir := conf.Storage.TargetDirectory
	doi := metadata.Identifier.ID

	err := os.MkdirAll(filepath.Join(prepdir, doi), os.ModePerm)
	if err != nil {
		log.Print("Could not create the preparation directory")
		return err
	}
	err = os.MkdirAll(filepath.Join(storagedir, doi), os.ModePerm)
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
	log.Printf("Cloning %s to directory %s", URI, destdir)

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
	// fail registration on missing datacite.yaml file
	dataciteText, err := readFileAtURL(repoFileURL(conf, repository, "datacite.yml"))
	if err != nil {
		// Can happen if the datacite.yml file is removed and the user clicks the register button on a stale page
		err := fmt.Errorf("%s <p><i>No datacite.yml file found in repository</i></p>", msgInvalidDOI)
		return nil, err
	}

	// fail registration on invalid datacite.yaml file
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

	// fail registration if unsupported values have been used
	if msgs := validateDataCiteValues(repoMetadata); len(msgs) > 0 {
		err := fmt.Errorf("%s<i><p>%s</p></i>", msgInvalidDOI, strings.Join(msgs, "</p><p>"))
		return nil, err
	}

	return repoMetadata, nil
}

// getPreviousDOI checks if the repository to be registered has a fork with a
// registered DOI under the service's user, which indicates that it already has
// been registered and this is a new version of the same dataset. If at any
// point it fails with an error, it logs the error and returns an empty string.
func getPreviousDOI(job *RegistrationJob) string {
	// We could infer the repository's fork path by replacing the owner in the
	// string with 'doi' (or the service's user), but it might be the case that
	// a DOI owned repository already exists with the same name and is *not* a
	// fork of this one (repo name collision).
	client := job.Config.GIN.Session
	repo := job.Metadata.SourceRepository
	forks, err := getRepoForks(client, repo)
	if err != nil {
		return ""
	}
	for _, fork := range forks {
		if strings.ToLower(fork.Owner.UserName) == client.Username {
			// fork owned by DOI user: Check for tags
			prevDOI, err := getLatestDOITag(client, &fork, job.Config.DOIBase)
			if err != nil {
				return ""
			}
			return prevDOI
		}
	}
	return ""
}

// getRepoForks returns a list of forks for the repository.
func getRepoForks(client *ginclient.Client, repo string) ([]gogs.Repository, error) {
	reqpath := fmt.Sprintf("api/v1/repos/%s/forks", repo)
	resp, err := client.Get(reqpath)
	if err != nil {
		log.Printf("Failed get forks for %q: %s", repo, err.Error())
		return nil, err
	}
	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Printf("Failed read forks from response for %q: %s", repo, err.Error())
		return nil, err
	}
	forks := make([]gogs.Repository, 0)
	err = json.Unmarshal(data, &forks)
	if err != nil {
		log.Printf("Failed to unmarshal forks for %q: %s", repo, err.Error())
	}
	return forks, err
}

// getLatestDOITag returns the most recent repository tag that matches our DOI
// prefix.
func getLatestDOITag(client *ginclient.Client, repo *gogs.Repository, doiBase string) (string, error) {
	// NOTE: The following API endpoint isn't available on GIN, but it has been
	// added to GOGS upstream. This wont work until we update GIN Web.
	reqpath := fmt.Sprintf("api/v1/repos/%s/releases", repo.FullName)
	resp, err := client.Get(reqpath)
	if err != nil {
		log.Printf("Failed to get releases for %q: %s", repo.FullName, err.Error())
		return "", err
	}
	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Printf("Failed to read releases from response for %q: %s", repo.FullName, err.Error())
		return "", err
	}
	tags := make([]gogs.Release, 0)
	err = json.Unmarshal(data, &tags)
	if err != nil {
		log.Printf("Failed to unmarshal releases for %q: %s", repo.FullName, err.Error())
		return "", err
	}
	var latestTime int64
	latestTag := ""
	for _, tag := range tags {
		if strings.Contains(tag.Name, doiBase) && libgin.IsRegisteredDOI(tag.Name) {
			tagTime := tag.Created.Unix()
			if tagTime > latestTime {
				latestTag = tag.Name
				latestTime = tagTime
			}
		}
	}
	return latestTag, nil
}

// MakeZip recursively writes all the files found under the provided sources to
// the dest io.Writer in ZIP format.  Any directories listed in source are
// archived recursively.  Empty directories and directories and files specified
// via the exclude parameter are ignored.
// The zip file has no compression by design since most zipped files are large
// binary files that do not compress well, while it might take a decent amount
// of time in addition.
func MakeZip(dest io.Writer, exclude []string, source ...string) error {
	// NOTE: Does not support commits other than master.

	// check sources
	for _, src := range source {
		if _, err := os.Stat(src); err != nil {
			return fmt.Errorf("Cannot access '%s': %s", src, err.Error())
		}
	}

	zipwriter := zip.NewWriter(dest)
	defer zipwriter.Close()

	walker := func(path string, fi os.FileInfo, err error) error {

		// return on any error
		if err != nil {
			return err
		}

		// return with specific SkipDir error when encountering an excluded directory or file;
		// if it is a direcory, the directory content will be excluded as well.
		for i := range exclude {
			if exclude[i] == path {
				return filepath.SkipDir
			}
		}

		// create a new dir/file header
		header, err := zip.FileInfoHeader(fi)
		if err != nil {
			return err
		}

		// update the name to correctly reflect the desired destination when unzipping
		// header.Name = strings.TrimPrefix(strings.Replace(file, src, "", -1), string(filepath.Separator))
		header.Name = path

		if fi.Mode().IsDir() {
			return nil
		}

		// write the header
		w, err := zipwriter.CreateHeader(header)
		if err != nil {
			return err
		}

		// Dereference symlinks
		if fi.Mode()&os.ModeSymlink != 0 {
			data, err := os.Readlink(path)
			if err != nil {
				return err
			}
			if _, err := io.Copy(w, strings.NewReader(data)); err != nil {
				return err
			}
			return nil
		}

		// open files for zipping
		f, err := os.Open(path)
		defer f.Close()
		if err != nil {
			return err
		}

		// copy file data into zip writer
		if _, err := io.Copy(w, f); err != nil {
			return err
		}

		return nil
	}

	// walk path
	for _, src := range source {
		err := filepath.Walk(src, walker)
		if err != nil {
			return fmt.Errorf("Error adding %s to zip file: %s", src, err.Error())
		}
	}
	return nil
}
