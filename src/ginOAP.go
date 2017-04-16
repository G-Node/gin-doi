package ginDoi

import (
	"encoding/json"
	"fmt"
	log "github.com/Sirupsen/logrus"
	"io/ioutil"
	"net/http"
	"github.com/G-Node/gin-core/gin"
	"bytes"
)

var (
	gOAPLOGP = "GinOAP"
)

type GinOauthProvider struct {
	Name     string
	Uri      string
	ApiKey   string
	KeyURL   string
	TokenURL string
}

func (pr *GinOauthProvider) ValidateToken(userName string, token string) (bool, error) {
	resp, err := http.Get(fmt.Sprintf(pr.TokenURL, token))
	if err != nil {
		log.WithFields(log.Fields{
			"source": gOAPLOGP,
			"error":  err,
		}).Debug("Token Validation failed")
		return false, err
	}
	if resp.StatusCode != http.StatusOK{
		return false, nil
	}
	return true, nil
}

func (pr *GinOauthProvider) getUser(userName string, token string) (OauthIdentity, error) {
	client := &http.Client{}
	req, _ := http.NewRequest("GET", fmt.Sprintf("%s/%s", pr.Uri, userName), nil)
	req.Header.Set("Authorization", token)
	resp, err := client.Do(req)
	if err != nil {
		log.WithFields(log.Fields{
			"source": gOAPLOGP,
			"error":  err,
		}).Debug("Authorisation server reponse malformed")
		return OauthIdentity{}, err
	}
	if resp.StatusCode != http.StatusOK {
		log.WithFields(log.Fields{
			"source": gOAPLOGP,
			"error":  err,
		}).Debug("Authorisation server reponse malformed")
		return OauthIdentity{}, fmt.Errorf("[%s] Server reponse malformed", gOAPLOGP)
	}
	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.WithFields(log.Fields{
			"source": gOAPLOGP,
			"error":  err,
		}).Debug("Could not read body from auth server")
		return OauthIdentity{}, err
	}
	user := OauthIdentity{}
	if err := json.Unmarshal(data, &user); err != nil {
		log.WithFields(log.Fields{
			"source": gOAPLOGP,
			"error":  err,
		}).Debug("Could not unmarshal user profile")
		return OauthIdentity{}, err
	}

	return user, err
}

func (pr *GinOauthProvider) AuthorizePull(user OauthIdentity, key gin.SSHKey) (error) {
	cl := http.Client{}
	bd, _ := json.Marshal(key)
	req, _ := http.NewRequest(http.MethodPost, fmt.Sprintf(pr.KeyURL, user.Login), bytes.NewReader(bd))
	resp, err := cl.Do(req)
	if err != nil {
		log.WithFields(log.Fields{
			"source": gOAPLOGP,
			"error":  err,
		}).Error("Could not put ssh key in server")
		return err
	}
	if resp.StatusCode != http.StatusOK {
		log.WithFields(log.Fields{
			"source":   gOAPLOGP,
			"Response": resp,
		}).Error("Could not put ssh key in server")
		return fmt.Errorf("Could not put ssh key")
	}
	return nil
}

func (pr *GinOauthProvider) DeAuthorizePull(user OauthIdentity, key gin.SSHKey) (error) {
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
