package ginDoi

import (
	"encoding/json"
	"fmt"
	log "github.com/Sirupsen/logrus"
	"io/ioutil"
	"net/http"
)

var (
	gOAPLOGP = "GinOAP"
)

func (pr *OauthProvider) getUser(userName string, token string) (OauthIdentity, error) {
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

	if len(user.EmailRaw) > 0 {
		return user, err
	} else {
		return user, fmt.Errorf("User not Authenticated")
	}
}
