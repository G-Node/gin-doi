package main

import (
	"fmt"
	"net/http"

	log "github.com/sirupsen/logrus"
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
	req.Header.Set("Cookie", fmt.Sprintf("i_like_gogs=%s", token))
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
