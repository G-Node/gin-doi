package ginDoi

import (
	"testing"
	"net/http/httptest"
	"net/http"
	"io/ioutil"
	"reflect"
)
func TestDoiGet(t *testing.T){
	mock :=`doi=12345
url=12345`
	t.SkipNow()
	srv := httptest.NewServer(http.HandlerFunc(
	func(w http.ResponseWriter, r *http.Request){
		bd,_ := ioutil.ReadAll(r.Body)
		bdTxt := string(bd)
		if !reflect.DeepEqual(bdTxt, mock) {
			t.Logf("[DoiP Err]%s equal %s:%s", bdTxt, mock,reflect.DeepEqual(bdTxt, mock))
			t.Fail()
		}
		w.WriteHeader(http.StatusCreated)
	}))
	defer srv.Close()

	dp := DoiProvider{ApiURI: srv.URL, Pwd:""}
	re, err := dp.RegDoi(DoiInfo{Authors:[]string{"Christian Garbers"}, Title:"New title",
			DOI:"12345"})
	if err != nil{
		t.Logf("[DoiP Err] Error was :%s",err)
		t.Fail()
	} else{
		t.Logf("[Ok] %s", re)
	}
}
