package main

import (
	"bytes"
	"crypto/rsa"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/G-Node/gin-core/gin"
	log "github.com/Sirupsen/logrus"
	"golang.org/x/crypto/ssh"
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
	AvatarURL string `json:"avatar_url"`
}

type GogsOAuthProvider struct {
	Name     string
	URI      string
	APIKey   string
	KeyURL   string
	TokenURL string
}

type GogsPublicKey struct {
	Key   string `json:"key"`
	Title string `json:"title,omitempty"`
}

func (pr *GogsOAuthProvider) ValidateToken(userName string, token string) (bool, error) {
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
			"source":  gogsOAPLOGP,
			"token":   token,
			"request": req,
		}).Debug("Token Validation failed")
		return false, nil
	}
	return true, nil
}

func (pr *GogsOAuthProvider) getUser(userName string, token string) (OAuthIdentity, error) {
	client := &http.Client{}
	req, _ := http.NewRequest("GET", fmt.Sprintf("%s", pr.URI), nil)
	req.Header.Set("Cookie", fmt.Sprintf("i_like_gogits=%s", token))
	resp, err := client.Do(req)
	if err != nil {
		log.WithFields(log.Fields{
			"source": gogsOAPLOGP,
			"error":  err,
		}).Debug("Authorisation server response malformed")
		return OAuthIdentity{}, err
	}
	if resp.StatusCode != http.StatusOK {
		log.WithFields(log.Fields{
			"source":  gogsOAPLOGP,
			"request": req,
		}).Debug("Authorisation server response malformed")
		return OAuthIdentity{}, fmt.Errorf("[%s] Server response malformed", gogsOAPLOGP)
	}
	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.WithFields(log.Fields{
			"source": gogsOAPLOGP,
			"error":  err,
		}).Debug("Could not read body from auth server")
		return OAuthIdentity{}, err
	}
	gogsuser := gogsUser{}
	if err = json.Unmarshal(data, &gogsuser); err != nil {
		log.WithFields(log.Fields{
			"source": gogsOAPLOGP,
			"error":  err,
		}).Debug("Could not unmarshal user profile")
		return OAuthIdentity{}, err
	}
	log.WithFields(log.Fields{
		"User": gogsuser,
	}).Debug("User")
	user := OAuthIdentity{}
	user.Token = token
	user.Login = gogsuser.UserName
	user.LastName = gogsuser.FullName
	user.UUID = fmt.Sprintf("fromgogs: %d", gogsuser.ID)
	user.Email = &gin.Email{}
	user.Email.Email = gogsuser.Email
	return user, err
}

func (pr *GogsOAuthProvider) AuthorizePull(user OAuthIdentity) (*rsa.PrivateKey, error) {
	rsaKey, err := genSSHKey()
	if err != nil {
		return nil, err
	}
	pub, err := ssh.NewPublicKey(&rsaKey.PublicKey)
	if err != nil {
		return nil, err
	}
	key := GogsPublicKey{Key: string(ssh.MarshalAuthorizedKey(pub)), Title: ssh.FingerprintSHA256(pub)}
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

func (pr *GogsOAuthProvider) DeAuthorizePull(user OAuthIdentity, key gin.SSHKey) error {
	return nil
}
