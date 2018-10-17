package main

import (
	"bytes"
	"net/http"
	"path/filepath"
	"text/template"

	log "github.com/Sirupsen/logrus"
)

const LOGPREFIX = "GnodeDOIProvider"

type GnodeDOIProvider struct {
	//https://mds.datacite.org/static/apidoc
	APIURI  string
	Pwd     string
	DOIBase string
}

func MakeDOI(UUID, DOIBase string) string {
	return DOIBase + UUID[:6]
}

func (dp GnodeDOIProvider) MakeDOI(doiInfo *CBerry) string {
	doiInfo.DOI = MakeDOI(doiInfo.UUID[:6], dp.DOIBase)
	return doiInfo.DOI
}

func (dp GnodeDOIProvider) GetXML(doiInfo *CBerry) (string, error) {
	dp.MakeDOI(doiInfo)
	t, err := template.ParseFiles(filepath.Join("tmpl", "datacite.xml"))
	if err != nil {
		log.WithFields(log.Fields{
			"source": LOGPREFIX,
			"error":  err,
		}).Error("Could not parse template")
		return "", err
	}
	buff := bytes.Buffer{}
	err = t.Execute(&buff, doiInfo)
	if err != nil {
		log.WithFields(log.Fields{
			"source": LOGPREFIX,
			"error":  err,
		}).Error("Template execution failed")
		return "", err
	}
	return buff.String(), err
}

func (dp GnodeDOIProvider) RegDOI(doiInfo CBerry) (string, error) {
	data, err := dp.GetXML(&doiInfo)
	if err != nil {
		return "", err
	}
	if r, err := http.Post(dp.APIURI+"/metadata", "text/plain;charset=UTF-8",
		bytes.NewReader([]byte(data))); err != nil {
		return "", err
	} else {
		return r.Status, nil
	}
}
