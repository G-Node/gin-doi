package ginDoi

import (
	"crypto/rsa"
	"github.com/G-Node/gin-core/gin"
)

var (
	MS_NODOIFILE      = "Could not locate a cloudberry file. Please visit https://web.gin.g-node.org/info/doi for a guide"
	MS_INVALIDDOIFILE = "The doi File was not Valid. Please visit https://web.gin.g-node.org/info/doi for a guide"
	MS_URIINVALID     = "Please provide a valid repository URI"
	MS_SERVERWORKS    = "The doi server has started doifying you repository. " +
		"Once finnished it will be availible <a href=\"%s\" class=\"label label-warning\">here</a>. Please return to that location to check for " +
		"availibility <br><br>" +
		"We will try to resgister the following doi: <div class =\"label label-default\">%s</div> " +
		"for your dataset. Please note, however, that in rare cases the final doi might be different."
	MS_NOLOGIN        = "You are not logged in with the gin service. Login at http://gin.g-node.org/"
	MS_NOTOKEN        = "No authentication token provided"
	MS_NOUSER         = "No username provided"
	MS_NOTITLE        = "No Title provided."
	MS_NOAUTHORS      = "No Authors provided."
	MS_AUTHORWRONG    = "Not all Authors valid.  Please provide at least a lastname and a firstname"
	MS_NODESC         = "No Description provided."
	MS_NOLIC          = "No Valid Liecense provided.Plaese specify url and name!"
	MS_REFERENCEWRONG = "A specified Reference is not valid (needs name and type)"
	DSOURCELOGPREFIX  = "DataSource"
	GINREPODOIPATH    = "/users/%s/repos/%s/browse/master/cloudberry.yml"
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
	Mess       string
	DoiInfo    CBerry
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
