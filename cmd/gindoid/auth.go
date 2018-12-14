package main

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/G-Node/gin-core/gin"
	log "github.com/sirupsen/logrus"
	"golang.org/x/crypto/ssh"
)

// User represents a API user.
type User struct {
	ID        int64  `json:"id"`
	UserName  string `json:"login"`
	FullName  string `json:"full_name"`
	Email     string `json:"email"`
	AvatarURL string `json:"avatar_url"`
}

// OAuthProvider represents an authentication server.
type OAuthProvider struct {
	Name     string
	URI      string
	APIKey   string
	KeyURL   string
	TokenURL string
}

// PublicKey is a public SSH key and an optional title (description).
type PublicKey struct {
	Key   string `json:"key"`
	Title string `json:"title,omitempty"`
}

// ValidateToken checks if the given token is valid for the user by making an authenticated request to the OAuthProvider.
func (pr *OAuthProvider) ValidateToken(userName string, token string) (bool, error) {
	client := &http.Client{}
	req, _ := http.NewRequest("GET", fmt.Sprintf("%s", pr.KeyURL), nil)
	req.Header.Set("Cookie", fmt.Sprintf("i_like_gogits=%s", token))
	resp, err := client.Do(req)
	if err != nil {
		log.WithFields(log.Fields{
			"source": lpAuth,
			"error":  err,
		}).Error("Token Validation failed")
		return false, err
	}
	if resp.StatusCode != http.StatusOK {
		log.WithFields(log.Fields{
			"source":  lpAuth,
			"token":   token,
			"request": req,
		}).Debug("Token Validation failed")
		return false, nil
	}
	return true, nil
}

func (pr *OAuthProvider) getUser(userName string, token string) (OAuthIdentity, error) {
	client := &http.Client{}
	req, _ := http.NewRequest("GET", fmt.Sprintf("%s", pr.URI), nil)
	req.Header.Set("Cookie", fmt.Sprintf("i_like_gogits=%s", token))
	resp, err := client.Do(req)
	if err != nil {
		log.WithFields(log.Fields{
			"source": lpAuth,
			"error":  err,
		}).Debug("Authorisation server response malformed")
		return OAuthIdentity{}, err
	}
	if resp.StatusCode != http.StatusOK {
		log.WithFields(log.Fields{
			"source":  lpAuth,
			"request": req,
		}).Debug("Authorisation server response malformed")
		return OAuthIdentity{}, fmt.Errorf("[%s] Server response malformed", lpAuth)
	}
	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.WithFields(log.Fields{
			"source": lpAuth,
			"error":  err,
		}).Debug("Could not read body from auth server")
		return OAuthIdentity{}, err
	}
	gogsuser := User{}
	if err = json.Unmarshal(data, &gogsuser); err != nil {
		log.WithFields(log.Fields{
			"source": lpAuth,
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

func (pr *OAuthProvider) AuthorizePull(user OAuthIdentity) (*rsa.PrivateKey, error) {
	rsaKey, err := genSSHKey()
	if err != nil {
		return nil, err
	}
	pub, err := ssh.NewPublicKey(&rsaKey.PublicKey)
	if err != nil {
		return nil, err
	}

	title := fmt.Sprintf("GIN DOI: %s", ssh.FingerprintSHA256(pub))
	key := PublicKey{Key: string(ssh.MarshalAuthorizedKey(pub)), Title: title}
	log.WithFields(log.Fields{
		"source": lpAuth,
		"Key":    key,
	}).Debug("About to send Key")
	cl := http.Client{}
	bd, err := json.Marshal(key)
	if err != nil {
		return nil, err
	}
	log.WithFields(log.Fields{
		"source":        lpAuth,
		"MarshalledKey": string(bd),
	}).Debug("About to send Marshalled Key")
	req, err := http.NewRequest(http.MethodPost, fmt.Sprintf(pr.KeyURL), bytes.NewReader(bd))
	req.Header.Set("Cookie", fmt.Sprintf("i_like_gogits=%s", user.Token))
	req.Header.Set("content-type", "application/json")
	if err != nil {
		log.WithFields(log.Fields{
			"source": lpAuth,
			"error":  err,
		}).Error("Could not Create Post request to post ssh key")
		return nil, err
	}
	resp, err := cl.Do(req)
	if err != nil {
		log.WithFields(log.Fields{
			"source":   lpAuth,
			"error":    err,
			"Response": resp,
			"Request":  req,
		}).Error("Could not put ssh key to server")
		return nil, err
	}
	if resp.StatusCode != http.StatusCreated {
		data, _ := ioutil.ReadAll(resp.Body)
		log.WithFields(log.Fields{
			"source":   lpAuth,
			"Response": resp,
			"Request":  req,
			"Body":     string(data),
		}).Error("Could not put ssh key to server")
		return nil, fmt.Errorf("Could not put ssh key")
	}
	return rsaKey, nil
}

func (pr *OAuthProvider) DeAuthorizePull(user OAuthIdentity, key gin.SSHKey) error {
	return nil
}

func genSSHKey() (*rsa.PrivateKey, error) {
	rsaKey, err := rsa.GenerateKey(rand.Reader, 1024)
	if err != nil {
		return nil, err
	}
	return rsaKey, nil
}
