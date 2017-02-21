package ginDoi

import (
	"net/http"
	"bytes"
	"fmt"
)

type DoiProvider struct {
	//https://mds.datacite.org/static/apidoc
	ApiURI string
}

func (dp *DoiProvider) RegDoi(target string) (string, error){
	bd := fmt.Sprintf("doi=12345\nurl=%s", target)
	if r,err :=http.Post(dp.ApiURI,"text/plain;charset=UTF-8", bytes.NewBufferString(bd));err != nil{
		return "", err
	}else {
		return r.Status, nil
	}
}