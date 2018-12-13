package main

import (
	"io/ioutil"
	"net/http/httptest"
	"strings"
	"testing"

	log "github.com/sirupsen/logrus"
)

type MockStorage struct {
	LocalStorage
}

func getInit(querrySting string, ds DataSource, pr OAuthProvider) string {
	rec := httptest.NewRecorder()
	// req := httptest.NewRequest(http.MethodPost, "/"+querrySting, bytes.NewReader([]byte("")))
	// InitDOIJob(rec, req, ds, pr, "../tmpl")
	body, _ := ioutil.ReadAll(rec.Body)
	return string(body)
}

//https://doi.gin.g-node.org/?repo=master%3Atesti%2Ftest&user=testi&token=123"
func TestInitDOIJob(t *testing.T) {
	log.SetLevel(log.DebugLevel)
	if !strings.Contains(getInit("", &GogsDataSource{}, nil), MS_URIINVALID) {
		t.Log("[Err] No URI should complain")
		t.Fail()
		return
	}
	if !strings.Contains(getInit("?repo=master", &GogsDataSource{}, nil),
		MS_NOTOKEN) {
		t.Log("[Err] No Token should complain")
		t.Fail()
		return
	}
	if !strings.Contains(getInit("?repo=master&token=123", &GogsDataSource{},
		nil), MS_NOUSER) {
		t.Log("[Err] No User should complain")
		t.Fail()
		return
	}
	if !strings.Contains(getInit("?repo=master&token=Bearer%20123&user=chris", &GogsDataSource{},
		MockOAuthProvider{ValidToken: false}), MS_NOLOGIN) {
		t.Log("[Err] No valid token should complain")
		t.Fail()
		return
	}

	if !strings.Contains(getInit("?repo=master&token=Bearer%20123&user=chris",
		&MockDataSource{validDOIFile: false, Berry: DOIRegInfo{Missing: []string{"sads"}}},
		MockOAuthProvider{ValidToken: true}), MS_INVALIDDOIFILE) {
		t.Log("[Err] No valid doifile should complain")
		t.Fail()
		return
	}
	t.Log("[Ok] Init DOI Job")
	return

}
