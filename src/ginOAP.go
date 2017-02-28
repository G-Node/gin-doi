package ginDoi

import (
	"net/http"
	"log"
	"io/ioutil"
	"encoding/json"
	"fmt"
)

var (
	gOAPLOGP = "GinOAP"
)

func (pr * OauthProvider) getUser(userName string, token string) (OauthIdentity, error) {
	client := &http.Client{}
	req, _ := http.NewRequest("GET", fmt.Sprintf("%s/%s", pr.Uri, userName), nil)
	req.Header.Set("Authorization", token)
	resp, err:= client.Do(req)
	log.Printf("[%s] Request: %+v",gOAPLOGP, req)
	log.Printf("[%s] Response Header: %v",gOAPLOGP, resp.StatusCode)
	if err != nil{
		log.Printf("[%s] Server reponse malformed:%s", gOAPLOGP, err)
		return OauthIdentity{}, err
	}
	if resp.StatusCode != http.StatusOK  {
		log.Printf("[%s] Server reponse malformed", gOAPLOGP)
		return OauthIdentity{},fmt.Errorf("[%s] Server reponse malformed", gOAPLOGP)
	}
	data, err := ioutil.ReadAll(resp.Body)
	if err != nil{
		log.Printf("[%s] Could not read Body from Server:%s", gOAPLOGP, err)
		return OauthIdentity{}, err
	}
	user := OauthIdentity{}
	if err := json.Unmarshal(data, &user); err != nil{
		log.Printf("[%s] Could not unmarshal user Profile:%s", gOAPLOGP, err)
		return OauthIdentity{}, err
	}

	if len(user.EmailRaw) > 0 {
		return user, err
	}else {
		return user, fmt.Errorf("User not Authenticated")
	}
}
