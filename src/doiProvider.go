package ginDoi

import (
	"net/http"
	"bytes"
	"text/template"
	"log"
)

var LOGPREFIX = "DoiProvider"
type DoiProvider struct {
	//https://mds.datacite.org/static/apidoc
	ApiURI string
	Pwd string
	DOIBase string
}

func (dp *DoiProvider) MakeDoi(doiInfo *CBerry) string {
	doiInfo.DOI = dp.DOIBase + "/" + doiInfo.UUID[:10]
	return doiInfo.DOI
}

func (dp *DoiProvider) GetXml(doiInfo *CBerry) ([]byte, error) {
	dp.MakeDoi(doiInfo)
	t, err := template.ParseFiles("tmpl/datacite.xml")
	if err != nil{
		log.Printf("[%s] Template broken:%s", LOGPREFIX, err)
		return nil, err
	}
	buff := bytes.Buffer{}
	err = t.Execute(&buff,doiInfo)
	if err != nil{
		log.Printf("[%s] template execution failed:%s", LOGPREFIX, err)
		return nil, err
	}
	return buff.Bytes(), err
}

func (dp *DoiProvider) RegDoi(doiInfo CBerry) (string, error){
	data, err := dp.GetXml(&doiInfo)
	if err != nil{
		return "",err
	}
	if r,err :=http.Post(dp.ApiURI+"/metadata","text/plain;charset=UTF-8", bytes.NewReader(data));err != nil{
		return "", err
	}else {
		return r.Status, nil
	}
}