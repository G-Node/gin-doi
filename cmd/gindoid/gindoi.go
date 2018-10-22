package main

import (
	"crypto/rsa"
	"html/template"
	"regexp"

	"github.com/G-Node/gin-core/gin"
)

const (
	MS_NODOIFILE      = `Could not locate a datacite file. Please check <a href="https://web.gin.g-node.org/G-Node/Info/wiki/DOIfile">here</a> for detailed instructions. `
	MS_INVALIDDOIFILE = `The doi File was not valid. Please check <a href="https://web.gin.g-node.org/G-Node/Info/wiki/DOIfile">here</a> for detailed instructions. `
	MS_URIINVALID     = "Please provide a valid repository URI"
	MS_DOIREG         = `<i class="info icon"></i>
						<div class="content">
							<div class="header"> A DOI is already registered for your dataset.</div>
							Your DOI is: <br>
								<div class ="ui label label-default"><a href="https://doi.org/%s">%s</a>
							</div>.
						</div>`
	MS_SERVERWORKS = `<i class="notched circle loading icon"></i>
		<div class="content">
			<div class="header">The DOI server has started archiving your repository.</div>
		We will try to register the following DOI for your dataset:<br>
		<div class ="ui label label-default"><a href="https://doi.org/%s">%s</a></div><br>
		In rare cases the final DOI might be different.<br>
		Please note that the final step in the registration process requires us to manually review your request.
		It may therefore take a few hours until the DOI is finally registered and your data becomes available.
		We will notify you via email once the process is finished.<br>
		<b>This page can safely be closed. You do not need to keep it open.</b>
		</div>`
	MS_NOLOGIN        = `You are not logged in with the gin service. Login <a href="http://gin.g-node.org/">here</a>`
	MS_NOTOKEN        = "No authentication token provided"
	MS_NOUSER         = "No username provided"
	MS_NOTITLE        = "No title provided."
	MS_NOAUTHORS      = "No authors provided."
	MS_AUTHORWRONG    = "Not all authors valid.  Please provide at least a lastname and a firstname"
	MS_NODESC         = "No description provided."
	MS_NOLIC          = "No valid license provided. Please specify url and name!"
	MS_REFERENCEWRONG = "A specified Reference is not valid (needs name and type)"
	DSOURCELOGPREFIX  = "DataSource"
	GINREPODOIPATH    = "/users/%s/repos/%s/browse/master/datacite.yml"
	MS_ENCODING       = `There was an issue with the content of your doifile. This might mean that the encoding is wrong.
						Please check <a href="https://web.gin.g-node.org/G-Node/Info/wiki/DOIfile">here</a> for detailed instructions or write an email to gin@g-node.org`
)

// StorageElement is responsible for storing elements defined by source to a kind of Storage
// defined by target
type StorageElement interface {
	// Should return true if the target location is already there
	Exists(target string) (bool, error)
	// Store the things specified by source in target
	Put(source string, target string) (bool, error)
	GetDataSource() (*DataSource, error)
}

type OAuthProvider interface {
	ValidateToken(userName string, token string) (bool, error)
	getUser(userName string, token string) (OAuthIdentity, error)
	AuthorizePull(user OAuthIdentity) (*rsa.PrivateKey, error)
	DeAuthorizePull(user OAuthIdentity, key gin.SSHKey) error
}

type Storage interface {
	Put(job DOIJob) error
	GetDataSource() (*DataSource, error)
}

type DataSource interface {
	ValidDOIFile(URI string, user OAuthIdentity) (bool, *DOIRegInfo)
	CloneRepository(URI string, To string, key *rsa.PrivateKey, hostsfile string) (string, error)
	MakeUUID(URI string, user OAuthIdentity) (string, error)
}

type DOIProvider interface {
	MakeDOI(doiInfo *DOIRegInfo) string
	GetXML(doiInfo *DOIRegInfo, doixml string) (string, error)
	RegDOI(doiInfo DOIRegInfo, doixml string) (string, error)
}

type DOIUser struct {
	Name       string
	Identities []OAuthIdentity
	MainOId    OAuthIdentity
}

type DOIReq struct {
	URI        string
	User       DOIUser
	OAuthLogin string
	Token      string
	Message    template.HTML
	DOIInfo    *DOIRegInfo
}

type OAuthIdentity struct {
	gin.Account
	Token string
}

// DOIJob holds the attributes needed to perform unit of work.
type DOIJob struct {
	Name    string
	Source  string
	Storage LocalStorage
	User    OAuthIdentity
	Request DOIReq
	Key     rsa.PrivateKey
}

func (d *DOIReq) GetDOIURI() string {
	var re = regexp.MustCompile(`(.+)\/`)
	return string(re.ReplaceAll([]byte(d.URI), []byte("doi/")))

}

func (d *DOIReq) AsHTML() template.HTML {
	return template.HTML(d.Message)
}
