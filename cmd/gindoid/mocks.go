package main

import (
	"crypto/rsa"
	"fmt"
	"os"

	"github.com/G-Node/gin-core/gin"
)

type MockDataSource struct {
	calls        []string
	validDOIFile bool
	Berry        DOIRegInfo
}

func (ds *MockDataSource) ValidDOIFile(URI string, user OAuthIdentity) (bool, *DOIRegInfo) {
	return ds.validDOIFile, &ds.Berry
}
func (ds *MockDataSource) CloneRepository(URI string, To string, key *rsa.PrivateKey, hostsfile string) (string, error) {
	os.Mkdir(To, os.ModePerm)
	ds.calls = append(ds.calls, fmt.Sprintf("%s, %s", URI, To))
	return "", nil
}

func (ds *MockDataSource) MakeUUID(URI string, user OAuthIdentity) (string, error) {
	return "123", nil
}

type MockDOIProvider struct {
}

func (dp MockDOIProvider) MakeDOI(doiInfo *DOIRegInfo) string {
	return "133"
}
func (dp MockDOIProvider) GetXML(doiInfo *DOIRegInfo, doixml string) (string, error) {
	return "xml", nil
}
func (dp MockDOIProvider) RegDOI(doiInfo DOIRegInfo) (string, error) {
	return "", nil
}

type MockOAuthProvider struct {
	ValidToken bool
	User       OAuthIdentity
}

func (op MockOAuthProvider) ValidateToken(userName string, token string) (bool, error) {
	return op.ValidToken, nil
}

func (op MockOAuthProvider) getUser(userName string, token string) (OAuthIdentity, error) {
	return op.User, nil
}

func (op MockOAuthProvider) AuthorizePull(user OAuthIdentity) (*rsa.PrivateKey, error) {
	return &rsa.PrivateKey{}, nil
}

func (op MockOAuthProvider) DeAuthorizePull(user OAuthIdentity, key gin.SSHKey) error {
	return nil
}
