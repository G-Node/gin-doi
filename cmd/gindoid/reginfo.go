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
	"time"

	"github.com/G-Node/libgin/libgin"
	yaml "gopkg.in/yaml.v2"
)

// DOIMData holds all the metadata for a dataset that's in the process of being registered.
type DOIMData struct {
	Data struct {
		ID         string `json:"id"`
		Type       string `json:"type"`
		Attributes struct {
			DOI        string      `json:"doi"`
			Identifier string      `json:"identifier"`
			URL        interface{} `json:"url"`
			Author     []struct {
				Literal string `json:"literal"`
			} `json:"author"`
			Title               string      `json:"title"`
			ContainerTitle      string      `json:"container-title"`
			Description         string      `json:"description"`
			ResourceTypeSubtype string      `json:"resource-type-subtype"`
			DataCenterID        string      `json:"data-center-id"`
			MemberID            string      `json:"member-id"`
			ResourceTypeID      string      `json:"resource-type-id"`
			Version             string      `json:"version"`
			License             interface{} `json:"license"`
			SchemaVersion       string      `json:"schema-version"`
			Results             []struct {
				ID    string `json:"id"`
				Title string `json:"title"`
				Count int    `json:"count"`
			} `json:"results"`
			RelatedIdentifiers []struct {
				RelationTypeID    string `json:"relation-type-id"`
				RelatedIdentifier string `json:"related-identifier"`
			} `json:"related-identifiers"`
			Published  string      `json:"published"`
			Registered time.Time   `json:"registered"`
			Updated    time.Time   `json:"updated"`
			Media      interface{} `json:"media"`
			XML        string      `json:"xml"`
		} `json:"attributes"`
		Relationships struct {
			DataCenter struct {
				Meta struct {
				} `json:"meta"`
			} `json:"data-center"`
			Member struct {
				Meta struct {
				} `json:"meta"`
			} `json:"member"`
			ResourceType struct {
				Meta struct {
				} `json:"meta"`
			} `json:"resource-type"`
		} `json:"relationships"`
	} `json:"data"`
}

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

// parseDOIInfo parses the DOI registration info and returns a filled DOIRegInfo struct.
func parseDOIInfo(infoyml []byte) (*libgin.DOIRegInfo, error) {
	doiInfo := libgin.DOIRegInfo{}
	err := yaml.Unmarshal(infoyml, &doiInfo)
	if err != nil {
		log.Print("Could not unmarshal DOI file")
		res := libgin.DOIRegInfo{}
		res.Missing = []string{fmt.Sprintf("%s", err)}
		return &res, fmt.Errorf("error while unmarshalling DOI info: %s", err.Error())
	}
	doiInfo.DateTime = time.Now()
	if !hasValues(&doiInfo) {
		log.Print("DOI file is missing entries")
		return &doiInfo, fmt.Errorf("DOI info is missing entries")
	}
	return &doiInfo, nil
}

type Author struct {
	FirstName   string
	LastName    string
	Affiliation string
	ID          string
}

type NamedIdentifier struct {
	URI    string
	Scheme string
	ID     string
}

func (a *Author) GetValidID() *NamedIdentifier {
	if a.ID == "" {
		return nil
	}
	if strings.Contains(strings.ToLower(a.ID), "orcid") {
		// assume the orcid id is a four block number thing eg. 0000-0002-5947-9939
		var re = regexp.MustCompile(`(\d+-\d+-\d+-\d+)`)
		nid := string(re.Find([]byte(a.ID)))
		return &NamedIdentifier{URI: "https://orcid.org/", Scheme: "ORCID", ID: nid}
	}
	return nil
}
func (a *Author) RenderAuthor() string {
	auth := fmt.Sprintf("%s,%s;%s;%s", a.LastName, a.FirstName, a.Affiliation, a.ID)
	return strings.TrimRight(auth, ";")
}

type Reference struct {
	Reftype  string
	Name     string
	Citation string
	ID       string
}

func (ref Reference) GetURL() string {
	idparts := strings.SplitN(ref.ID, ":", 2)
	if len(idparts) != 2 {
		// Malformed ID (no colon)
		return ""
	}
	source := idparts[0]
	idnum := idparts[1]

	var prefix string
	switch strings.ToLower(source) {
	case "doi":
		prefix = "https://doi.org/"
	case "arxiv":
		// https://arxiv.org/help/arxiv_identifier_for_services
		prefix = "https://arxiv.org/abs/"
	case "pmid":
		// https://www.ncbi.nlm.nih.gov/books/NBK3862/#linkshelp.Retrieve_PubMed_Citations
		prefix = "https://www.ncbi.nlm.nih.gov/pubmed/"
	default:
		// Return an empty string to make the reflink inactive
		return ""
	}

	return fmt.Sprintf("%s%s", prefix, idnum)
}

type License struct {
	Name string
	URL  string
}

func hasValues(s *libgin.DOIRegInfo) bool {
	if s.Title == "" {
		s.Missing = append(s.Missing, msgNoTitle)
	}
	if len(s.Authors) == 0 {
		s.Missing = append(s.Missing, msgNoAuthors)
	} else {
		for _, auth := range s.Authors {
			if auth.LastName == "" || auth.FirstName == "" {
				s.Missing = append(s.Missing, msgInvalidAuthors)
			}
		}
	}
	if s.Description == "" {
		s.Missing = append(s.Missing, msgNoDescription)
	}
	if s.License == nil || s.License.Name == "" || s.License.URL == "" {
		s.Missing = append(s.Missing, msgNoLicense)
	}
	if s.References != nil {
		for _, ref := range s.References {
			if (ref.Citation == "" && ref.Name == "") || ref.Reftype == "" {
				s.Missing = append(s.Missing, msgInvalidReference)
			}
		}
	}
	return len(s.Missing) == 0
}

type DOIReq struct {
	RequestData string
	*libgin.DOIRequestData
	Message       template.HTML
	DOIInfo       *libgin.DOIRegInfo
	ErrorMessages []string
}

func (d *DOIReq) GetDOIURI() string {
	var re = regexp.MustCompile(`(.+)\/`)
	return string(re.ReplaceAll([]byte(d.Repository), []byte("doi/")))
}

func (d *DOIReq) AsHTML() template.HTML {
	return template.HTML(d.Message)
}

// renderXML creates the DataCite XML file contents given the registration data and XML template.
func renderXML(doiInfo *libgin.DOIRegInfo) (string, error) {
	tmpl, err := txttemplate.New("doixml").Parse(doiXML)
	if err != nil {
		log.Print("Could not parse template")
		return "", err
	}
	buff := bytes.Buffer{}
	err = tmpl.Execute(&buff, doiInfo)
	if err != nil {
		log.Print("Template execution failed")
		return "", err
	}
	return buff.String(), err
}
