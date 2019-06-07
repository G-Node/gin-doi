package main

import (
	"bytes"
	"text/template"

	log "github.com/sirupsen/logrus"
)

func GetXML(doiInfo *DOIRegInfo, doixml string) (string, error) {
	t, err := template.ParseFiles(doixml)
	if err != nil {
		log.WithFields(log.Fields{
			"source": lpMakeXML,
			"error":  err,
		}).Error("Could not parse template")
		return "", err
	}
	buff := bytes.Buffer{}
	err = t.Execute(&buff, doiInfo)
	if err != nil {
		log.WithFields(log.Fields{
			"source": lpMakeXML,
			"error":  err,
		}).Error("Template execution failed")
		return "", err
	}
	return buff.String(), err
}
