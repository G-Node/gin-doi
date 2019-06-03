package main

import (
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/G-Node/gin-cli/ginclient"
	"github.com/G-Node/gin-cli/ginclient/config"
	"github.com/G-Node/gin-cli/git"
	"github.com/G-Node/gin-cli/git/shell"
	gogs "github.com/gogits/go-gogs-client"
	log "github.com/sirupsen/logrus"
	yaml "gopkg.in/yaml.v2"
)

type DataSource struct {
	pubKey       string
	session      *ginclient.Client
	ServerConfig *config.ServerCfg
	Username     string
	Password     string
}

// GinURL returns the full URL for the configured GIN server, constructed from
// the server configuration.
func (s *DataSource) GinURL() string {
	return s.ServerConfig.Web.AddressStr()
}

// GitURL returns the full git URL for the configured GIN server, constructed
// from the server configuration.
func (s *DataSource) GitURL() string {
	return s.ServerConfig.Git.AddressStr()
}

func (s *DataSource) getDOIFile(URI string, user OAuthIdentity) ([]byte, error) {
	// git archive --remote=git://git.foo.com/project.git HEAD:path/to/directory filename
	// https://github.com/go-yaml/yaml.git
	// git@github.com:go-yaml/yaml.git
	// TODO: config variables for path etc.
	fetchRepoPath := fmt.Sprintf("%s/raw/master/datacite.yml", URI)
	client := &http.Client{}
	req, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("%s/%s", s.GinURL(), fetchRepoPath), nil)
	req.Header.Add("Cookie", fmt.Sprintf("i_like_gogits=%s", user.Token))
	resp, err := client.Do(req)
	if err != nil {
		// todo Try to infer what went wrong
		log.WithFields(log.Fields{
			"path":   fetchRepoPath,
			"source": lpDataSource,
			"error":  err,
		}).Debug("Could not get DOI file")
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("could not get DOI file: %s", resp.Status)
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.WithFields(log.Fields{
			"path":   fetchRepoPath,
			"source": lpDataSource,
			"error":  err,
		}).Debug("Could not read from received datacite.yml file")
		return nil, err
	}
	return body, nil
}

func (s *DataSource) Login() error {
	hostkeystr, _, err := git.GetHostKey(s.ServerConfig.Git)
	if err != nil {
		return fmt.Errorf("Failed to get host key during server setup: %v", err)
	}
	s.ServerConfig.Git.HostKey = hostkeystr
	err = config.AddServerConf("gin", *s.ServerConfig)
	if err != nil {
		return fmt.Errorf("Failed to set up server configuration: %v", err)
	}

	gincl := ginclient.New("gin")
	err = gincl.Login(s.Username, s.Password, "gin-doi")
	if err != nil {
		gerr := err.(shell.Error)
		log.Error(gerr.Origin)
		log.Error(gerr.UError)
		log.Error(gerr.Description)
		return err
	}
	s.session = gincl
	return nil
}

// CloneRepo clones a git repository (with git-annex) specified by URI to the
// destination directory.
func (s *DataSource) CloneRepo(URI string, destdir string) error {
	// NOTE: CloneRepo changes the working directory to the cloned repository
	// See: https://github.com/G-Node/gin-cli/issues/225
	// This will need to change when that issue is fixed
	origdir, err := os.Getwd()
	if err != nil {
		log.Errorf("%s: Failed to get working directory when cloning repository. Was our working directory removed?", lpStorage)
		return err
	}
	defer os.Chdir(origdir)
	err = os.Chdir(destdir)
	if err != nil {
		return err
	}
	log.WithFields(log.Fields{
		"URI":     URI,
		"session": s.session,
		"source":  lpDataSource,
	}).Debug("Start cloning")

	clonechan := make(chan git.RepoFileStatus)
	go s.session.CloneRepo(strings.ToLower(URI), clonechan)
	for stat := range clonechan {
		log.Debug(stat)
		if stat.Err != nil {
			log.Errorf("Repository cloning failed: %s", stat.Err)
			return stat.Err
		}
	}

	downloadchan := make(chan git.RepoFileStatus)
	go s.session.GetContent(nil, downloadchan)
	for stat := range downloadchan {
		log.Debug(stat)
		if stat.Err != nil {
			log.Errorf("Repository cloning failed during annex get: %s", stat.Err)
			return stat.Err
		}
	}
	return nil
}

var UUIDMap = map[string]string{
	"INT/multielectrode_grasp":                   "f83565d148510fede8a277f660e1a419",
	"ajkumaraswamy/HB-PAC_disinhibitory_network": "1090f803258557299d287c4d44a541b2",
	"steffi/Kleineidam_et_al_2017":               "f53069de4c4921a3cfa8f17d55ef98bb",
	"Churan/Morris_et_al_Frontiers_2016":         "97bc1456d3f4bca2d945357b3ec92029",
	"fabee/efish_locking":                        "6953bbf0087ba444b2d549b759de4a06",
}

func RepoP2UUID(URI string) string {
	if doi, ok := UUIDMap[URI]; ok {
		return doi
	}
	currMd5 := md5.Sum([]byte(URI))
	return hex.EncodeToString(currMd5[:])
}
func (s *DataSource) MakeUUID(URI string, user OAuthIdentity) (string, error) {
	return RepoP2UUID(URI), nil
}

// GetMasterCommit determines the latest commit id of the master branch
func (s *DataSource) GetMasterCommit(URI string, user OAuthIdentity) (string, error) {
	fetchRepoPath := fmt.Sprintf("%s", URI)
	client := &http.Client{}
	req, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("%s/api/v1/repos/%s/branches", s.GinURL(), fetchRepoPath), nil)
	req.Header.Add("Cookie", fmt.Sprintf("i_like_gogits=%s", user.Token))
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("Could not get repo branches: %s", resp.Status)
	}

	branches := []gogs.Branch{}
	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	json.Unmarshal(data, &branches)
	for _, branch := range branches {
		if branch.Name == "master" {
			return branch.Commit.ID, nil
		}
	}
	return "", fmt.Errorf("Could not locate master branch")
}

// ValidDOIFile returns true if the specified URI has a DOI file containing all necessary information.
func (s *DataSource) ValidDOIFile(URI string, user OAuthIdentity) (bool, *DOIRegInfo) {
	in, err := s.getDOIFile(URI, user)
	if err != nil {
		log.WithFields(log.Fields{
			"data":   string(in),
			"source": lpDataSource,
			"error":  err,
		}).Error("Could not get the DOI file")
		return false, nil
	}
	doiInfo := DOIRegInfo{}
	err = yaml.Unmarshal(in, &doiInfo)
	if err != nil {
		log.WithFields(log.Fields{
			"data":   string(in),
			"source": lpDataSource,
			"error":  err,
		}).Error("Could not unmarshal DOI file")
		res := DOIRegInfo{}
		res.Missing = []string{fmt.Sprintf("%s", err)}
		return false, &res
	}
	doiInfo.DateTime = time.Now()
	if !hasValues(&doiInfo) {
		log.WithFields(log.Fields{
			"data":    string(in),
			"doiInfo": doiInfo,
			"source":  lpDataSource,
			"error":   err,
		}).Debug("DOI file is missing entries")
		return false, &doiInfo
	}
	return true, &doiInfo
}

type DOIRegInfo struct {
	Missing      []string
	DOI          string
	UUID         string
	FileSize     int64
	Title        string
	Authors      []Author
	Description  string
	Keywords     []string
	References   []Reference
	Funding      []string
	License      *License
	ResourceType string
	DateTime     time.Time
}

func (c *DOIRegInfo) GetType() string {
	if c.ResourceType != "" {
		return c.ResourceType
	}
	return "Dataset"
}

func (c *DOIRegInfo) GetCitation() string {
	var authors string
	for _, auth := range c.Authors {
		if len(auth.FirstName) > 0 {
			authors += fmt.Sprintf("%s %s, ", auth.LastName, string(auth.FirstName[0]))
		} else {
			authors += fmt.Sprintf("%s, ", auth.LastName)
		}
	}
	return fmt.Sprintf("%s (%s) %s. G-Node. doi:%s", authors, c.Year(), c.Title, c.DOI)
}

func (c *DOIRegInfo) EscXML(txt string) string {
	buf := new(bytes.Buffer)
	if err := xml.EscapeText(buf, []byte(txt)); err != nil {
		log.Errorf("Could not escape:%s, %+v", txt, err)
		return ""
	}
	return buf.String()
}

func (c *DOIRegInfo) Year() string {
	return fmt.Sprintf("%d", c.DateTime.Year())
}

func (c *DOIRegInfo) ISODate() string {
	return c.DateTime.Format("2006-01-02")
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

func (c *Author) GetValidID() *NamedIdentifier {
	if c.ID == "" {
		return nil
	}
	if strings.Contains(strings.ToLower(c.ID), "orcid") {
		// assume the orcid id is a four block number thing eg. 0000-0002-5947-9939
		var re = regexp.MustCompile(`(\d+-\d+-\d+-\d+)`)
		nid := string(re.Find([]byte(c.ID)))
		return &NamedIdentifier{URI: "https://orcid.org/", Scheme: "ORCID", ID: nid}
	}
	return nil
}
func (a *Author) RenderAuthor() string {
	auth := fmt.Sprintf("%s,%s;%s;%s", a.LastName, a.FirstName, a.Affiliation, a.ID)
	return strings.TrimRight(auth, ";")
}

type Reference struct {
	Reftype string
	Name    string
	ID      string
}

func (ref Reference) GetURL() string {
	idparts := strings.SplitN(ref.ID, ":", 2)
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

func hasValues(s *DOIRegInfo) bool {
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
			if ref.Name == "" || ref.Reftype == "" {
				s.Missing = append(s.Missing, msgInvalidReference)
			}
		}
	}
	return len(s.Missing) == 0
}
