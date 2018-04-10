package ginDoi

import (
	"crypto/rsa"
	"github.com/G-Node/gin-core/gin"
	"regexp"
	"html/template"
)

var (
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
	MS_SERVERWORKS    = `<i class="notched circle loading icon"></i>
		<div class="content">
			<div class="header">The doi server has started doifying and archiving your repository.</div>
		We will try to register the following doi:<br>
		<div class ="ui label label-default"><a href="https://doi.org/%s">%s</a></div><br>
		for your dataset. Please note, however, that in rare cases the final doi might be different.<br>
		Please consider that there is a step of human intervention before the doi is registered.
		It might therefore take a few hours until the doi page goes live. We will notify you
		via email once the process is finished.<br>
		<b>This page can safely be closed. You do not need to keep it open</b>
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

// Responsible for storing smth defined by source to a kind of Storage
// defined by target
type StorageElement interface {
	// Should return true if the target location is alredy there
	Exists(target string) (bool, error)
	// Store the things specifies by source in target
	Put(source string, target string) (bool, error)
	GetDataSource() (*DataSource, error)
}

type OauthProvider interface {
	ValidateToken(userName string, token string) (bool, error)
	getUser(userName string, token string) (OauthIdentity, error)
	AuthorizePull(user OauthIdentity) (*rsa.PrivateKey, error)
	DeAuthorizePull(user OauthIdentity, key gin.SSHKey) (error)
}

type Storage interface {
	Put(job DoiJob) error
	GetDataSource() (*DataSource, error)
}

type DataSource interface {
	ValidDoiFile(URI string, user OauthIdentity) (bool, *CBerry)
	Get(URI string, To string, key *rsa.PrivateKey) (string, error)
	MakeUUID(URI string, user OauthIdentity) (string, error)
}

type DoiProvider interface {
	MakeDoi(doiInfo *CBerry) string
	GetXml(doiInfo *CBerry) (string, error)
	RegDoi(doiInfo CBerry) (string, error)
}

type DoiUser struct {
	Name       string
	Identities []OauthIdentity
	MainOId    OauthIdentity
}

type DoiReq struct {
	URI        string
	User       DoiUser
	OauthLogin string
	Token      string
	Mess       template.HTML
	DoiInfo    *CBerry
}

type OauthIdentity struct {
	gin.Account
	Token string
}

// DoiJob holds the attributes needed to perform unit of work.
type DoiJob struct {
	Name    string
	Source  string
	Storage LocalStorage
	User    OauthIdentity
	DoiReq  DoiReq
	Key     rsa.PrivateKey
}


func (d *DoiReq) GetdoiUri() string {
	var re = regexp.MustCompile(`(.+)\/`)
	return string(re.ReplaceAll([]byte(d.URI),[]byte("doi/")))

}

func (d *DoiReq) AsHtml() template.HTML {
	return template.HTML(d.Mess)
}