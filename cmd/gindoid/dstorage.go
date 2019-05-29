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
	repopath := job.Source
	jobname := job.Name
	dReq := &job.Request

	ls.prepDir(jobname, dReq.DOIInfo)

	targetpath := filepath.Join(ls.Path, jobname)
	preperrors := make([]string, 0, 5)
	zipsize, err := ls.cloneandzip(repopath, jobname, targetpath)
	if err != nil {
		// failed to clone and zip
		// save the error for reporting and continue with the XML prep
		preperrors = append(preperrors, err.Error())
	}
	ls.createIndexFile(jobname, dReq)
	dReq.DOIInfo.FileSize = zipsize/(1024*1000) + 1 // Proper size conversion to closest human-readable size

	fp, err := os.Create(filepath.Join(targetpath, "doi.xml"))
	if err != nil {
		log.WithFields(log.Fields{
			"source": lpStorage,
			"error":  err,
			"target": jobname,
		}).Error("Could not create the metadata template")
		preperrors = append(preperrors, fmt.Sprintf("Failed to create the XML metadata template: %s", err))
	}
	defer fp.Close()

	// No registering. But the XML is provided with everything
	doixml := filepath.Join(ls.TemplatePath, doixmlfname)
	data, err := GetXML(dReq.DOIInfo, doixml)
	if err != nil {
		log.WithFields(log.Fields{
			"source": lpStorage,
			"error":  err,
			"target": jobname,
		}).Error("Could not parse the metadata file")
		preperrors = append(preperrors, fmt.Sprintf("Failed to parse the XML metadata: %s", err))
	}
	_, err = fp.Write([]byte(data))
	if err != nil {
		log.WithFields(log.Fields{
			"source": lpStorage,
			"error":  err,
			"target": jobname,
		}).Error("Could not write to the metadata file")
		preperrors = append(preperrors, fmt.Sprintf("Failed to write the metadata XML file: %s", err))
	}
	dReq.ErrorMessages = preperrors
	ls.sendMaster(dReq)
	return err
}

// cloneandzip clones the source repository into a temporary directory under targetpath, zips the contents, and returns the size of the zip file in bytes.
func (ls LocalStorage) cloneandzip(repopath string, jobname string, targetpath string) (int64, error) {
	// Clone under a tmp/ subdirectory of the zip target path
	clonetmp := filepath.Join(targetpath, tmpdir)
	if err := os.MkdirAll(clonetmp, 0777); err != nil {
		errmsg := fmt.Sprintf("Failed to create temporary clone directory: %s", tmpdir)
		log.Error(errmsg)
		return -1, fmt.Errorf(errmsg)
	}

	// Clone
	ds := ls.GetDataSource()
	if err := ds.CloneRepo(repopath, clonetmp); err != nil {
		log.WithFields(log.Fields{
			"source": lpStorage,
			"error":  err,
			"target": jobname,
		}).Error("Repository cloning failed")
		return -1, fmt.Errorf("Failed to clone repository '%s': %s", repopath, err)
	}

	// Zip
	zipsize, err := ls.zip(jobname)
	if err != nil {
		log.WithFields(log.Fields{
			"source": lpStorage,
			"error":  err,
			"target": jobname,
		}).Error("Could not zip the data")
		return -1, fmt.Errorf("Failed to create the zip file: %s", err)
	}
	return zipsize, nil
}

func (ls *LocalStorage) zip(target string) (int64, error) {
	// filepath.Abs only returns error if the CWD doesn't exist, so we can
	// safely ignore it here
	destdir, err := filepath.Abs(filepath.Join(ls.Path, target))
	if err != nil {
		log.Errorf("%s: Failed to get abs path for destination directory (%s, %s) while making ZIP file. Was our working directory removed?", lpStorage, ls.Path, target)
		return 0, err
	}
	srcdir := filepath.Join(destdir, tmpdir)
	log.WithFields(log.Fields{
		"source":  lpStorage,
		"destdir": destdir,
	}).Debug("Started zipping")
	fp, err := os.Create(filepath.Join(destdir, target+".zip"))
	if err != nil {
		log.WithFields(log.Fields{
			"source":  lpStorage,
			"error":   err,
			"destdir": destdir,
		}).Error("Could not create zip file")
		return 0, err
	}
	defer fp.Close()

	// Change into clone directory to make the paths in the zip archive repo
	// root relative.
	origdir, err := os.Getwd()
	if err != nil {
		log.Errorf("%s: Failed to get working directory when making ZIP file. Was our working directory removed?", lpStorage)
		return 0, err
	}
	defer os.Chdir(origdir)
	os.Chdir(srcdir)

	err = libgin.MakeZip(fp, ".")
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
	tmpl, err := template.ParseFiles(filepath.Join(ls.TemplatePath, "doiInfo.tmpl"))
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
			"source":  lpStorage,
			"error":   err,
			"doiInfo": info,
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
	repourl := fmt.Sprintf("%s/%s", ls.Source.GinURL(), repopath)

	errorlist := ""
	if len(dReq.ErrorMessages) > 0 {
		errorlist = "The following errors occurred during the dataset preparation\n"
		for idx, msg := range dReq.ErrorMessages {
			errorlist = fmt.Sprintf("%s	%d. %s\n", errorlist, idx+1, msg)
		}
	}

	subject := fmt.Sprintf("New DOI registration request: %s", repopath)

	body := `A new DOI registration request has been received.

	Repository: %s [%s]
	User: %s
	Email address: %s
	DOI XML: %s
	DOI target URL: %s
	UUID: %s

%s
`
	body = fmt.Sprintf(body, repopath, repourl, userlogin, useremail, xmlurl, doitarget, uuid, errorlist)
	return ls.MServer.SendMail(subject, body)
}
