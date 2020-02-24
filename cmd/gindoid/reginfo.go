package main

import (
	"bytes"
	"fmt"
	"html/template"
	"io/ioutil"
	"log"
	"net/http"
	"regexp"
	"strings"
	txttemplate "text/template"

	"github.com/G-Node/libgin/libgin"
	yaml "gopkg.in/yaml.v2"
)

// dataciteURL returns the full URL to a repository's datacite.yml file.
func dataciteURL(repopath string, conf *Configuration) string {
	fetchRepoPath := fmt.Sprintf("%s/raw/master/datacite.yml", repopath)
	url := fmt.Sprintf("%s/%s", conf.GIN.Session.WebAddress(), fetchRepoPath)
	return url
}

// readFileAtURL returns the contents of a file at a given URL.
func readFileAtURL(url string) ([]byte, error) {
	client := &http.Client{}
	log.Printf("Fetching file at %q", url)
	req, _ := http.NewRequest(http.MethodGet, url, nil)
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("Error during request to GIN: %s", err.Error())
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Could not get DOI file: %s", resp.Status)
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Print("Could not read from received datacite.yml file")
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
		return nil, fmt.Errorf("The following required entries are not set: %s", strings.Join(missing, ", "))
	}
	return yamlInfo, nil
}

// checkMissingValues returns the list of required fields that have no values set.
func checkMissingValues(info *libgin.RepositoryYAML) []string {
	missing := []string{}
	if info.Title == "" {
		missing = append(missing, msgNoTitle)
	}
	if len(info.Authors) == 0 {
		missing = append(missing, msgNoAuthors)
	} else {
		for _, auth := range info.Authors {
			if auth.LastName == "" || auth.FirstName == "" {
				missing = append(missing, msgInvalidAuthors)
			}
		}
	}
	if info.Description == "" {
		missing = append(missing, msgNoDescription)
	}
	if info.License == nil || info.License.Name == "" || info.License.URL == "" {
		missing = append(missing, msgNoLicense)
	}
	if info.References != nil {
		for _, ref := range info.References {
			if (ref.Citation == "" && ref.Name == "") || ref.RefType == "" {
				missing = append(missing, msgInvalidReference)
			}
		}
	}
	return missing
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
	// Unmarshalled data from the datacite.yml of the repository being registered.
	DOIInfo *libgin.RepositoryYAML
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

// renderXML creates the DataCite XML file contents given the registration data and XML template.
func renderXML(metadata *libgin.DataCite) (string, error) {
	tmplfuncs := txttemplate.FuncMap{
		"EscXML":               EscXML,
		"ReferenceDescription": ReferenceDescription,
		"ReferenceID":          ReferenceID,
		"ReferenceSource":      ReferenceSource,
		"FunderName":           FunderName,
		"AwardNumber":          AwardNumber,
		"AuthorBlock":          AuthorBlock,
		"JoinComma":            JoinComma,
	}
	tmpl, err := txttemplate.New("doixml").Funcs(tmplfuncs).Parse(doiXML)
	if err != nil {
		log.Printf("Error parsing doi.xml template: %s", err.Error())
		return "", err
	}
	buff := bytes.Buffer{}
	err = tmpl.Execute(&buff, metadata)
	if err != nil {
		log.Printf("Error rendering doi.xml: %s", err.Error())
		return "", err
	}
	return buff.String(), err
}
