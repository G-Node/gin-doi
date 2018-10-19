package main

import (
	"fmt"
	"html/template"
	"os"
	"path/filepath"

	log "github.com/Sirupsen/logrus"
)

const (
	logprefix = "Storage"
	tmpdir    = "tmp"
)

type LocalStorage struct {
	Path         string
	Source       DataSource
	DProvider    DOIProvider
	HTTPBase     string
	MServer      *MailServer
	TemplatePath string
	SCPURL       string
}

func (ls *LocalStorage) Exists(target string) (bool, error) {
	return false, nil
}

func (ls LocalStorage) Put(job DOIJob) error {
	source := job.Source
	target := job.Name
	dReq := &job.Request

	//todo do this better
	to := filepath.Join(ls.Path, target)
	tmpDir := filepath.Join(to, tmpdir)
	ls.prepDir(target, dReq.DOIInfo)
	ds, _ := ls.GetDataSource()

	if out, err := ds.Get(source, tmpDir, &job.Key); err != nil {
		log.WithFields(log.Fields{
			"source": logprefix,
			"error":  err,
			"out":    out,
			"target": target,
		}).Error("Could not Get the data")
	}
	fSize, err := ls.zip(target)
	if err != nil {
		log.WithFields(log.Fields{
			"source": logprefix,
			"error":  err,
			"target": target,
		}).Error("Could not zip the data")
	}
	// +1 to report something with small datsets
	dReq.DOIInfo.FileSize = fSize/(1024*1000) + 1
	ls.createIndexFile(target, dReq)

	fp, _ := os.Create(filepath.Join(to, "doi.xml"))
	if err != nil {
		log.WithFields(log.Fields{
			"source": logprefix,
			"error":  err,
			"target": target,
		}).Error("Could not create parse the metadata template")
	}
	defer fp.Close()
	// No registering. But the xml is provided with everything
	data, err := ls.DProvider.GetXML(dReq.DOIInfo)
	if err != nil {
		log.WithFields(log.Fields{
			"source": logprefix,
			"error":  err,
			"target": target,
		}).Error("Could not create the metadata file")
	}
	_, err = fp.Write([]byte(data))
	if err != nil {
		log.WithFields(log.Fields{
			"source": logprefix,
			"error":  err,
			"target": target,
		}).Error("Could not write to the metadata file")
	}
	ls.sendMaster(dReq)
	return err
}

func (ls *LocalStorage) zip(target string) (int64, error) {
	to := filepath.Join(ls.Path, target)
	log.WithFields(log.Fields{
		"source": logprefix,
		"to":     to,
	}).Debug("Started zipping")
	fp, err := os.Create(filepath.Join(to, target+".zip"))
	if err != nil {
		log.WithFields(log.Fields{
			"source": logprefix,
			"error":  err,
			"to":     to,
		}).Error("Could not create zip file")
		return 0, err
	}
	defer fp.Close()
	err = Zip(filepath.Join(to, tmpdir), fp)
	stat, _ := fp.Stat()
	return stat.Size(), err
}

func (ls LocalStorage) GetDataSource() (DataSource, error) {
	return ls.Source, nil
}

func (ls LocalStorage) createIndexFile(target string, info *DOIReq) error {
	tmpl, err := template.ParseFiles(filepath.Join(ls.TemplatePath, "doiInfo.html"))
	if err != nil {
		if err != nil {
			log.WithFields(log.Fields{
				"source": logprefix,
				"error":  err,
				"target": target,
			}).Error("Could not parse the DOI template")
			return err
		}
		return err
	}

	fp, err := os.Create(filepath.Join(ls.Path, target, "index.html"))
	if err != nil {
		log.WithFields(log.Fields{
			"source": logprefix,
			"error":  err,
			"target": target,
		}).Error("Could not create the DOI index.html")
		return err
	}
	defer fp.Close()
	if err := tmpl.Execute(fp, info); err != nil {
		log.WithFields(log.Fields{
			"source":   logprefix,
			"error":    err,
			"doiInfoo": info,
		}).Error("Could not execute the DOI template")
		return err
	}
	return nil
}

func (ls *LocalStorage) prepDir(target string, info *DOIRegInfo) error {
	err := os.Mkdir(filepath.Join(ls.Path, target), os.ModePerm)
	if err != nil {
		log.WithFields(log.Fields{
			"source": logprefix,
			"error":  err,
			"target": target,
		}).Error("Could not create the target directory")
		return err
	}
	// Deny access per default
	file, err := os.Create(filepath.Join(ls.Path, target, ".htaccess"))
	if err != nil {
		log.WithFields(log.Fields{
			"source": logprefix,
			"error":  err,
			"target": target,
		}).Error("Could not create .httaccess")
		return err
	}
	defer file.Close()
	// todo check
	_, err = file.Write([]byte("deny from all"))
	if err != nil {
		log.WithFields(log.Fields{
			"source": logprefix,
			"error":  err,
			"target": target,
		}).Error("Could not write to .httaccess")
		return err
	}
	return nil
}
func (ls LocalStorage) getSCP(dReq *DOIReq) string {
	return fmt.Sprintf("%s/%s/doi.xml", ls.SCPURL, dReq.DOIInfo.UUID)
}
func (ls LocalStorage) sendMaster(dReq *DOIReq) error {

	repopath := dReq.URI
	userlogin := dReq.User.MainOId.Login
	useremail := dReq.User.MainOId.Account.Email.Email
	xmlurl := ls.getSCP(dReq)
	uuid := dReq.DOIInfo.UUID
	doitarget := fmt.Sprintf("%s/%s", ls.HTTPBase, uuid)

	body := `Subject: New DOI registration request: %s

A new DOI registration request has been received.

	User: %s
	Email address: %s
	DOI XML: %s
	DOI target URL: %s
	UUID: %s
`
	body = fmt.Sprintf(body, repopath, userlogin, useremail, xmlurl, doitarget, uuid)
	return ls.MServer.SendMail(body)
}
