package ginDoi

import (
	"net/http"
	"bytes"
	"text/template"
	log "github.com/Sirupsen/logrus"
	"path/filepath"
)

var LOGPREFIX = "DoiProvider"
type DoiProvider struct {
	//https://mds.datacite.org/static/apidoc
	ApiURI string
	Pwd string
	DOIBase string
}

func (dp *DoiProvider) MakeDoi(doiInfo *CBerry) string {
	doiInfo.DOI = dp.DOIBase + "/" + "G-NODE." + doiInfo.UUID[:10]
	return doiInfo.DOI
}

func (dp *DoiProvider) GetXml(doiInfo *CBerry) ([]byte, error) {
	dp.MakeDoi(doiInfo)
	t, err := template.ParseFiles(filepath.Join("tmpl", "datacite.xml"))
	if err != nil{
		log.WithFields(log.Fields{
			"source": LOGPREFIX,
			"error":err,
		}).Error("Could not parse template")
		return nil, err
	}
	buff := bytes.Buffer{}
	err = t.Execute(&buff,doiInfo)
	if err != nil{
		log.WithFields(log.Fields{
			"source": LOGPREFIX,
			"error":err,
		}).Error("Template execution failed")
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