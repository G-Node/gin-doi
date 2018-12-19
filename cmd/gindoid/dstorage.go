package main

import (
	"fmt"
	"html/template"
	"net/url"
	"os"
	"path/filepath"

	"github.com/G-Node/libgin/libgin"
	log "github.com/sirupsen/logrus"
)

const (
	tmpdir      = "tmp"
	doixmlfname = "datacite.xml"
)

type LocalStorage struct {
	Path         string
	Source       DataSource
	HTTPBase     string
	MServer      *MailServer
	TemplatePath string
	SCPURL       string
	KnownHosts   string
}

func (ls *LocalStorage) Exists(target string) (bool, error) {
	return false, nil
}

func (ls LocalStorage) Put(job DOIJob) error {
	source := job.Source
	target := job.Name
	dReq := &job.Request

	to := filepath.Join(ls.Path, target)
	tmpDir := filepath.Join(to, tmpdir)
	ls.prepDir(target, dReq.DOIInfo)
	ds := ls.GetDataSource()

	preperrors := make([]string, 0, 5)

	if out, err := ds.CloneRepository(source, tmpDir, &job.Key, ls.KnownHosts); err != nil {
		log.WithFields(log.Fields{
			"source": lpStorage,
			"error":  err,
			"out":    out,
			"target": target,
		}).Error("Repository cloning failed")
		preperrors = append(preperrors, fmt.Sprintf("Failed to clone repository '%s'", source))
	}
	fSize, err := ls.zip(target)
	if err != nil {
		log.WithFields(log.Fields{
			"source": lpStorage,
			"error":  err,
			"target": target,
		}).Error("Could not zip the data")
		preperrors = append(preperrors, "Failed to create the zip file")
	}
	// +1 to report something with small datasets
	dReq.DOIInfo.FileSize = fSize/(1024*1000) + 1
	ls.createIndexFile(target, dReq)

	fp, _ := os.Create(filepath.Join(to, "doi.xml"))
	if err != nil {
		log.WithFields(log.Fields{
			"source": lpStorage,
			"error":  err,
			"target": target,
		}).Error("Could not create the metadata template")
		preperrors = append(preperrors, "Failed to create the XML metadata template")
	}
	defer fp.Close()
	// No registering. But the XML is provided with everything

	doixml := filepath.Join(ls.TemplatePath, doixmlfname)
	data, err := GetXML(dReq.DOIInfo, doixml)
	if err != nil {
		log.WithFields(log.Fields{
			"source": lpStorage,
			"error":  err,
			"target": target,
		}).Error("Could not parse the metadata file")
		preperrors = append(preperrors, "Failed to parse the XML metadata")
	}
	_, err = fp.Write([]byte(data))
	if err != nil {
		log.WithFields(log.Fields{
			"source": lpStorage,
			"error":  err,
			"target": target,
		}).Error("Could not write to the metadata file")
		preperrors = append(preperrors, "Failed to write the metadata XML file")
	}
	dReq.ErrorMessages = preperrors
	ls.sendMaster(dReq)
	return err
}

func (ls *LocalStorage) zip(target string) (int64, error) {
	to := filepath.Join(ls.Path, target)
	log.WithFields(log.Fields{
		"source": lpStorage,
		"to":     to,
	}).Debug("Started zipping")
	fp, err := os.Create(filepath.Join(to, target+".zip"))
	if err != nil {
		log.WithFields(log.Fields{
			"source": lpStorage,
			"error":  err,
			"to":     to,
		}).Error("Could not create zip file")
		return 0, err
	}
	defer fp.Close()
	err = libgin.MakeZip(fp, filepath.Join(to, tmpdir))
	if err != nil {
		log.Errorf("MakeZip failed: %s", err)
		return 0, err
	}
	stat, _ := fp.Stat()
	return stat.Size(), err
}

func (ls LocalStorage) GetDataSource() DataSource {
	return ls.Source
}

func (ls LocalStorage) createIndexFile(target string, info *DOIReq) error {
	tmpl, err := template.ParseFiles(filepath.Join(ls.TemplatePath, "doiInfo.html"))
	if err != nil {
		if err != nil {
			log.WithFields(log.Fields{
				"source": lpStorage,
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
			"source": lpStorage,
			"error":  err,
			"target": target,
		}).Error("Could not create the DOI index.html")
		return err
	}
	defer fp.Close()
	if err := tmpl.Execute(fp, info); err != nil {
		log.WithFields(log.Fields{
			"source":   lpStorage,
			"error":    err,
			"doiInfoo": info,
		}).Error("Could not execute the DOI template")
		return err
	}
	return nil
}

func (ls *LocalStorage) prepDir(target string, info *DOIRegInfo) error {
	err := os.MkdirAll(filepath.Join(ls.Path, target), os.ModePerm)
	if err != nil {
		log.WithFields(log.Fields{
			"source": lpStorage,
			"error":  err,
			"target": target,
		}).Error("Could not create the target directory")
		return err
	}
	// Deny access per default
	file, err := os.Create(filepath.Join(ls.Path, target, ".htaccess"))
	if err != nil {
		log.WithFields(log.Fields{
			"source": lpStorage,
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
			"source": lpStorage,
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
	urljoin := func(a, b string) string {
		fallback := fmt.Sprintf("%s/%s (fallback URL join)", a, b)
		base, err := url.Parse(a)
		if err != nil {
			return fallback
		}
		suffix, err := url.Parse(b)
		if err != nil {
			return fallback
		}
		return base.ResolveReference(suffix).String()
	}

	repopath := dReq.URI
	userlogin := dReq.User.MainOId.Login
	useremail := dReq.User.MainOId.Account.Email.Email
	xmlurl := ls.getSCP(dReq)
	uuid := dReq.DOIInfo.UUID
	doitarget := urljoin(ls.HTTPBase, uuid)

	errorlist := ""
	if len(dReq.ErrorMessages) > 0 {
		errorlist = "The following errors occurred during the dataset preparation\n"
		for idx, msg := range dReq.ErrorMessages {
			errorlist = fmt.Sprintf("%s	%d. %s\n", errorlist, idx+1, msg)
		}
	}

	subject := fmt.Sprintf("New DOI registration request: %s", repopath)

	body := `A new DOI registration request has been received.

	Repository: %s
	User: %s
	Email address: %s
	DOI XML: %s
	DOI target URL: %s
	UUID: %s

%s
`
	body = fmt.Sprintf(body, repopath, userlogin, useremail, xmlurl, doitarget, uuid, errorlist)
	return ls.MServer.SendMail(subject, body)
}
