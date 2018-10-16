package main

import (
	"fmt"
	"os"
	"crypto/rsa"
	"github.com/G-Node/gin-core/gin"
)

type MockDataSource struct {
	calls        []string
	validDoiFile bool
	Berry        CBerry
}

func (ds *MockDataSource) ValidDoiFile(URI string, user OauthIdentity) (bool, *CBerry) {
	return ds.validDoiFile, &ds.Berry
}
func (ds *MockDataSource) Get(URI string, To string, key *rsa.PrivateKey) (string, error) {
	os.Mkdir(To, os.ModePerm)
	ds.calls = append(ds.calls, fmt.Sprintf("%s, %s", URI, To))
	return "", nil
}

func (ds *MockDataSource) MakeUUID(URI string, user OauthIdentity) (string, error) {
	return "123", nil
}

type MockDoiProvider struct {
}

func (dp MockDoiProvider) MakeDoi(doiInfo *CBerry) string {
	return "133"
}
func (dp MockDoiProvider) GetXml(doiInfo *CBerry) (string, error) {
	return "xml", nil
}
func (dp MockDoiProvider) RegDoi(doiInfo CBerry) (string, error) {
	return "", nil
}

type MockOauthProvider struct {
	ValidToken bool
	User       OauthIdentity
}

func (op MockOauthProvider) ValidateToken(userName string, token string) (bool, error) {
	return op.ValidToken, nil
}

func (op MockOauthProvider) getUser(userName string, token string) (OauthIdentity, error) {
	return op.User, nil
}

func (op MockOauthProvider) AuthorizePull(user OauthIdentity) (*rsa.PrivateKey, error) {
	return &rsa.PrivateKey{}, nil
}

func (op MockOauthProvider) DeAuthorizePull(user OauthIdentity, key gin.SSHKey) (error) {
	return nil
}