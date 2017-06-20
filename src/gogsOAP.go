package ginDoi

import (
	"encoding/json"
	"fmt"
	log "github.com/Sirupsen/logrus"
	"io/ioutil"
	"net/http"
	"github.com/G-Node/gin-core/gin"
	"bytes"
	"crypto/rsa"
	"golang.org/x/crypto/ssh"
	"crypto/sha256"
	"strings"
	"encoding/base64"
)

var (
	gogsOAPLOGP = "GinOAP"
)

// User represents a API user.
type gogsUser struct {
	ID        int64  `json:"id"`
	UserName  string `json:"login"`
	FullName  string `json:"full_name"`
	Email     string `json:"email"`
	AvatarUrl string `json:"avatar_url"`
}

type GogsOauthProvider struct {
	Name     string
	Uri      string
	ApiKey   string
	KeyURL   string
	TokenURL string
}

type GogsPublicKey struct {
	Key   string    `json:"key"`
	Title string    `json:"title,omitempty"`
}

func (pr *GogsOauthProvider) ValidateToken(userName string, token string) (bool, error) {
	client := &http.Client{}
	req, _ := http.NewRequest("GET", fmt.Sprintf("%s", pr.KeyURL), nil)
	req.Header.Set("Cookie", fmt.Sprintf("i_like_gogits=%s", token))
	resp, err := client.Do(req)
	if err != nil {
		log.WithFields(log.Fields{
			"source": gogsOAPLOGP,
			"error":  err,
		}).Error("Token Validation failed")
		return false, err
	}
	if resp.StatusCode != http.StatusOK {
		log.WithFields(log.Fields{
			"source": gogsOAPLOGP,
			"token":  token,
			"request": req,
		}).Debug("Token Validation failed")
		return false, nil
	}
	return true, nil
}

func (pr *GogsOauthProvider) getUser(userName string, token string) (OauthIdentity, error) {
	client := &http.Client{}
	req, _ := http.NewRequest("GET", fmt.Sprintf("%s/%s", pr.Uri, userName), nil)
	req.Header.Set("Cookie", fmt.Sprintf("i_like_gogits=%s", token))
	resp, err := client.Do(req)
	if err != nil {
		log.WithFields(log.Fields{
			"source": gogsOAPLOGP,
			"error":  err,
		}).Debug("Authorisation server reponse malformed")
		return OauthIdentity{}, err
	}
	if resp.StatusCode != http.StatusOK {
		log.WithFields(log.Fields{
			"source":  gogsOAPLOGP,
			"request": req,
		}).Debug("Authorisation server reponse malformed")
		return OauthIdentity{}, fmt.Errorf("[%s] Server reponse malformed", gogsOAPLOGP)
	}
	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.WithFields(log.Fields{
			"source": gogsOAPLOGP,
			"error":  err,
		}).Debug("Could not read body from auth server")
		return OauthIdentity{}, err
	}
	gogsuser := gogsUser{}
	if err := json.Unmarshal(data, &gogsuser); err != nil {
		log.WithFields(log.Fields{
			"source": gogsOAPLOGP,
			"error":  err,
		}).Debug("Could not unmarshal user profile")
		return OauthIdentity{}, err
	}
	log.WithFields(log.Fields{
		"User": gogsuser,
	}).Debug("User")
	user := OauthIdentity{}
	user.Token = token
	user.Login = gogsuser.UserName
	user.LastName = gogsuser.FullName
	user.UUID = fmt.Sprintf("fromgogs:%s", gogsuser.ID)
	user.Email = &gin.Email{}
	user.Email.Email = gogsuser.Email
	return user, err
}

func (pr *GogsOauthProvider) AuthorizePull(user OauthIdentity) (*rsa.PrivateKey, error) {
	rsaKey, err := genSSHKey()
	if err != nil {
		return nil, err
	}
	pub, err := ssh.NewPublicKey(&rsaKey.PublicKey)
	if err != nil {
		return nil, err
	}
	key := GogsPublicKey{Key: string(ssh.MarshalAuthorizedKey(pub)), Title: FingerprintSHA256(pub)}
	log.WithFields(log.Fields{
		"source": gogsOAPLOGP,
		"Key":    key,
	}).Debug("About to send Key")
	cl := http.Client{}
	bd, err := json.Marshal(key)
	if err != nil {
		return nil, err
	}
	log.WithFields(log.Fields{
		"source":        gogsOAPLOGP,
		"MarshallesKey": string(bd),
	}).Debug("About to send Marshalled Key")
	req, err := http.NewRequest(http.MethodPost, fmt.Sprintf(pr.KeyURL), bytes.NewReader(bd))
	req.Header.Set("Cookie", fmt.Sprintf("i_like_gogits=%s", user.Token))
	req.Header.Set("content-type", "application/json")
	if err != nil {
		log.WithFields(log.Fields{
			"source": gogsOAPLOGP,
			"error":  err,
		}).Error("Could not Create Post request to post ssh key")
		return nil, err
	}
	resp, err := cl.Do(req)
	if err != nil {
		log.WithFields(log.Fields{
			"source":   gogsOAPLOGP,
			"error":    err,
			"Response": resp,
			"Request":  req,
		}).Error("Could not put ssh key to server")
		return nil, err
	}
	if resp.StatusCode != http.StatusCreated {
		data, _ := ioutil.ReadAll(resp.Body)
		log.WithFields(log.Fields{
			"source":   gogsOAPLOGP,
			"Response": resp,
			"Request":  req,
			"Body":     string(data),
		}).Error("Could not put ssh key to server")
		return nil, fmt.Errorf("Could not put ssh key")
	}
	return rsaKey, nil
}

func (pr *GogsOauthProvider) DeAuthorizePull(user OauthIdentity, key gin.SSHKey) (error) {
	return nil
}

//As Long as go does not ship it
func FingerprintSHA256(key ssh.PublicKey) string {
	hash := sha256.Sum256(key.Marshal())
	b64hash := base64.StdEncoding.EncodeToString(hash[:])
	return strings.TrimRight(b64hash, "=")
}
