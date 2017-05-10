package ginDoi

import (
	"fmt"
	"os"
	"crypto/rsa"
)

type MockDataSource struct {
	calls []string
}

func (ds *MockDataSource) ValidDoiFile(URI string, user OauthIdentity) (bool, *CBerry) {
	return true, &CBerry{}
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
