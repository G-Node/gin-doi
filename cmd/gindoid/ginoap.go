package main

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/G-Node/gin-core/gin"
	log "github.com/Sirupsen/logrus"
	"golang.org/x/crypto/ssh"
)

var (
	gOAPLOGP = "GinOAP"
)

type GinOAuthProvider struct {
	Name     string
	URI      string
	APIKey   string
	KeyURL   string
	TokenURL string
}

func (pr *GinOAuthProvider) ValidateToken(userName string, token string) (bool, error) {
	token = strings.Replace(token, "Bearer ", "", 1)
	resp, err := http.Get(fmt.Sprintf(pr.TokenURL, token))
	if err != nil {
		log.WithFields(log.Fields{
			"source": gOAPLOGP,
			"error":  err,
		}).Error("Token Validation failed")
		return false, err
	}
	if resp.StatusCode != http.StatusOK {
		log.WithFields(log.Fields{
			"source": gOAPLOGP,
			"token":  token,
		}).Debug("Token Validation failed")
		return false, nil
	}
	return true, nil
}

func (pr *GinOAuthProvider) getUser(userName string, token string) (OAuthIdentity, error) {
	client := &http.Client{}
	req, _ := http.NewRequest("GET", fmt.Sprintf("%s/%s", pr.URI, userName), nil)
	req.Header.Set("Authorization", token)
	resp, err := client.Do(req)
	if err != nil {
		log.WithFields(log.Fields{
			"source": gOAPLOGP,
			"error":  err,
		}).Debug("Authorisation server response malformed")
		return OAuthIdentity{}, err
	}
	if resp.StatusCode != http.StatusOK {
		log.WithFields(log.Fields{
			"source":  gOAPLOGP,
			"request": req,
		}).Debug("Authorisation server response malformed")
		return OAuthIdentity{}, fmt.Errorf("[%s] Server response malformed", gOAPLOGP)
	}
	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.WithFields(log.Fields{
			"source": gOAPLOGP,
			"error":  err,
		}).Debug("Could not read body from auth server")
		return OAuthIdentity{}, err
	}
	user := OAuthIdentity{}
	if err := json.Unmarshal(data, &user); err != nil {
		log.WithFields(log.Fields{
			"source": gOAPLOGP,
			"error":  err,
		}).Debug("Could not unmarshal user profile")
		return OAuthIdentity{}, err
	}
	user.Token = token
	return user, err
}

func (pr *GinOAuthProvider) AuthorizePull(user OAuthIdentity) (*rsa.PrivateKey, error) {
	rsaKey, err := genSSHKey()
	if err != nil {
		return nil, err
	}
	pub, err := ssh.NewPublicKey(&rsaKey.PublicKey)
	if err != nil {
		return nil, err
	}
	key := gin.SSHKey{Key: string(ssh.MarshalAuthorizedKey(pub)), Description: "Gin DOI Key"}
	cl := http.Client{}
	bd, err := json.Marshal(key)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest(http.MethodPost, fmt.Sprintf(pr.KeyURL, user.Login), bytes.NewReader(bd))
	req.Header.Set("Authorization", user.Token)
	if err != nil {
		log.WithFields(log.Fields{
			"source": gOAPLOGP,
			"error":  err,
		}).Error("Could not Create Post request to post ssh key")
		return nil, err
	}
	resp, err := cl.Do(req)
	if err != nil {
		log.WithFields(log.Fields{
			"source":   gOAPLOGP,
			"error":    err,
			"Response": resp,
			"Request":  req,
		}).Error("Could not put ssh key to server")
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		log.WithFields(log.Fields{
			"source":   gOAPLOGP,
			"Response": resp,
			"Request":  req,
		}).Error("Could not put ssh key to server")
		return nil, fmt.Errorf("Could not put ssh key")
	}
	return rsaKey, nil
}

func (pr *GinOAuthProvider) DeAuthorizePull(user OAuthIdentity, key gin.SSHKey) error {
	cl := http.Client{}
	bd, _ := json.Marshal(key)
	req, _ := http.NewRequest(http.MethodDelete, fmt.Sprintf(pr.KeyURL, user.Login), bytes.NewReader(bd))
	resp, err := cl.Do(req)
	if err != nil {
		log.WithFields(log.Fields{
			"source": gOAPLOGP,
			"error":  err,
		}).Error("Could not delete ssh key on server")
		return err
	}
	if resp.StatusCode != http.StatusOK {
		log.WithFields(log.Fields{
			"source":   gOAPLOGP,
			"Response": resp,
		}).Error("Could not delete ssh key in server")
		return fmt.Errorf("Could not put ssh key")
	}
	return nil
}

func genSSHKey() (*rsa.PrivateKey, error) {
	rsaKey, err := rsa.GenerateKey(rand.Reader, 1024)
	if err != nil {
		return nil, err
	}
	return rsaKey, nil
}
