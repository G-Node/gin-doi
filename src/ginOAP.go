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
	"crypto/rand"
	"strings"
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
	token = strings.Replace(token,"Bearer ","",1)
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

func (pr *GinOauthProvider) getUser(userName string, token string) (OauthIdentity, error) {
	client := &http.Client{}
	req, _ := http.NewRequest("GET", fmt.Sprintf("%s/%s", pr.Uri, userName), nil)
	req.Header.Set("Authorisation", token)
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
			"request":  req,
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

func (pr *GinOauthProvider) AuthorizePull(user OauthIdentity) (*rsa.PrivateKey, error) {
	rsaKey, err := genSSHKey()
	if err != nil {
		return nil, err
	}
	pub, err := ssh.NewPublicKey(&rsaKey.PublicKey)
	if err != nil {
		return nil, err
	}
	key := gin.SSHKey{Key: string(ssh.MarshalAuthorizedKey(pub)), Description: "Gin Doi Key"}
	cl := http.Client{}
	bd, err := json.Marshal(key)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest(http.MethodPost, fmt.Sprintf(pr.KeyURL, user.Login), bytes.NewReader(bd))
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
			"source": gOAPLOGP,
			"error":  err,
		}).Error("Could not put ssh key in server")
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		log.WithFields(log.Fields{
			"source":   gOAPLOGP,
			"Response": resp,
		}).Error("Could not put ssh key in server")
		return nil, fmt.Errorf("Could not put ssh key")
	}
	return rsaKey, nil
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

func genSSHKey() (*rsa.PrivateKey, error) {
	rsaKey, err := rsa.GenerateKey(rand.Reader, 1024)
	if err != nil {
		return nil, err
	}
	return rsaKey, nil
}
