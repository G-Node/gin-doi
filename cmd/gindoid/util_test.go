package main

import (
	"io/ioutil"
	"net/http/httptest"
	"strings"
	"testing"

	log "github.com/Sirupsen/logrus"
)

type MockStorage struct {
	LocalStorage
}

func getInit(querrySting string, ds DataSource, pr OauthProvider) string {
	rec := httptest.NewRecorder()
	// req := httptest.NewRequest(http.MethodPost, "/"+querrySting, bytes.NewReader([]byte("")))
	// InitDoiJob(rec, req, ds, pr, "../tmpl")
	body, _ := ioutil.ReadAll(rec.Body)
	return string(body)
}

//https://doi.gin.g-node.org/?repo=master%3Atesti%2Ftest&user=testi&token=123"
func TestInitDoiJob(t *testing.T) {
	log.SetLevel(log.DebugLevel)
	if !strings.Contains(getInit("", &GinDataSource{}, nil), MS_URIINVALID) {
		t.Log("[Err] No URI should complain")
		t.Fail()
		return
	}
	if !strings.Contains(getInit("?repo=master", &GinDataSource{}, nil),
		MS_NOTOKEN) {
		t.Log("[Err] No Token should complain")
		t.Fail()
		return
	}
	if !strings.Contains(getInit("?repo=master&token=123", &GinDataSource{},
		nil), MS_NOUSER) {
		t.Log("[Err] No User should complain")
		t.Fail()
		return
	}
	if !strings.Contains(getInit("?repo=master&token=Bearer%20123&user=chris", &GinDataSource{},
		MockOauthProvider{ValidToken: false}), MS_NOLOGIN) {
		t.Log("[Err] No valid token should complain")
		t.Fail()
		return
	}

	if !strings.Contains(getInit("?repo=master&token=Bearer%20123&user=chris",
		&MockDataSource{validDoiFile: false, Berry: CBerry{Missing: []string{"sads"}}},
		MockOauthProvider{ValidToken: true}), MS_INVALIDDOIFILE) {
		t.Log("[Err] No valid doifile should complain")
		t.Fail()
		return
	}
	t.Log("[Ok] Init Doi Job")
	return

}
