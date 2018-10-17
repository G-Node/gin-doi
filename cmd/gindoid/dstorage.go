package main

import (
	"fmt"
	"html/template"
	"io"
	"os"
	"path/filepath"
	txtTemplate "text/template"

	log "github.com/Sirupsen/logrus"
)

const (
	STORLOGPRE = "Storage"
	tmpdir     = "tmp"
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
			"source": STORLOGPRE,
			"error":  err,
			"out":    out,
			"target": target,
		}).Error("Could not Get the data")
	}
	fSize, err := ls.zip(target)
	if err != nil {
		log.WithFields(log.Fields{
			"source": STORLOGPRE,
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
			"source": STORLOGPRE,
			"error":  err,
			"target": target,
		}).Error("Could not create parse the metadata template")
	}
	defer fp.Close()
	// No registering. But the xml is provided with everything
	data, err := ls.DProvider.GetXML(dReq.DOIInfo)
	if err != nil {
		log.WithFields(log.Fields{
			"source": STORLOGPRE,
			"error":  err,
			"target": target,
		}).Error("Could not create the metadata file")
	}
	_, err = fp.Write([]byte(data))
	if err != nil {
		log.WithFields(log.Fields{
			"source": STORLOGPRE,
			"error":  err,
			"target": target,
		}).Error("Could not write to the metadata file")
	}
	ls.writeRegScript(to)
	ls.writeUpdIndexScript(to, dReq)
	ls.sendMaster(dReq)
	return err
}

func (ls *LocalStorage) zip(target string) (int64, error) {
	to := filepath.Join(ls.Path, target)
	log.WithFields(log.Fields{
		"source": STORLOGPRE,
		"to":     to,
	}).Debug("Started zipping")
	fp, err := os.Create(filepath.Join(to, target+".zip"))
	if err != nil {
		log.WithFields(log.Fields{
			"source": STORLOGPRE,
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

func (ls *LocalStorage) tar(target string) (int64, error) {
	to := filepath.Join(ls.Path, target)
	log.WithFields(log.Fields{
		"source": STORLOGPRE,
		"to":     to,
	}).Debug("Started taring")
	fp, err := os.Create(filepath.Join(to, target+".tar.gz"))
	if err != nil {
		log.WithFields(log.Fields{
			"source": STORLOGPRE,
			"error":  err,
			"to":     to,
		}).Error("Could not create zip file")
		return 0, err
	}
	defer fp.Close()
	err = Tar(filepath.Join(to, tmpdir), fp)
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
				"source": STORLOGPRE,
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
			"source": STORLOGPRE,
			"error":  err,
			"target": target,
		}).Error("Could not create the DOI index.html")
		return err
	}
	defer fp.Close()
	if err := tmpl.Execute(fp, info); err != nil {
		log.WithFields(log.Fields{
			"source":   STORLOGPRE,
			"error":    err,
			"doiInfoo": info,
		}).Error("Could not execute the DOI template")
		return err
	}
	return nil
}

func (ls *LocalStorage) prepDir(target string, info *CBerry) error {
	err := os.Mkdir(filepath.Join(ls.Path, target), os.ModePerm)
	if err != nil {
		log.WithFields(log.Fields{
			"source": STORLOGPRE,
			"error":  err,
			"target": target,
		}).Error("Could not create the target directory")
		return err
	}
	// Deny access per default
	file, err := os.Create(filepath.Join(ls.Path, target, ".htaccess"))
	if err != nil {
		log.WithFields(log.Fields{
			"source": STORLOGPRE,
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
			"source": STORLOGPRE,
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

	return ls.MServer.SendMail(
		fmt.Sprintf(
			`Hello. the fellowing Archives are ready for doification:%s. Creator:%s,%s
The DOI xml can be found here: %s. The DOI shall point to:%s/%s`,
			dReq.DOIInfo.UUID, dReq.User.MainOId.Account.Email.Email, dReq.User.MainOId.Login, ls.getSCP(dReq),
			ls.HTTPBase, dReq.DOIInfo.UUID))
}

func (ls LocalStorage) writeRegScript(target string) error {
	pScriptF, err := os.Open(filepath.Join(ls.TemplatePath, "mds-suite_test.pl"))
	if err != nil {
		log.WithFields(log.Fields{
			"source": STORLOGPRE,
			"error":  err,
			"target": target,
		}).Debug("The ugly Perls script is not there. Fuck it")
		return err
	}
	defer pScriptF.Close()

	pScriptT, err := os.Create(filepath.Join(target, "register.pl"))
	if err != nil {
		log.WithFields(log.Fields{
			"source": STORLOGPRE,
			"error":  err,
			"target": target,
		}).Debug("The ugly Perls script cannot be created. Screw it")
		return err
	}
	defer pScriptT.Close()

	_, err = io.Copy(pScriptT, pScriptF)
	if err != nil {
		log.WithFields(log.Fields{
			"source": STORLOGPRE,
			"error":  err,
			"target": target,
		}).Debug("The ugly Perl script cannot be written. HATE IT")
		return err
	}
	// todo error
	pScriptT.Chmod(0777)

	return err
}

func (ls LocalStorage) writeUpdIndexScript(target string, dReq *DOIReq) error {
	t, err := txtTemplate.ParseFiles(filepath.Join(ls.TemplatePath, "updIndex.sh"))
	if err != nil {
		log.WithFields(log.Fields{
			"source": STORLOGPRE,
			"error":  err,
			"target": target,
		}).Error("Could not parse the update index template")
		return err
	}
	fp, _ := os.Create(filepath.Join(target, "updIndex.sh"))
	if err != nil {
		log.WithFields(log.Fields{
			"source": STORLOGPRE,
			"error":  err,
			"target": target,
		}).Error("Could not create update index script")
		return err
	}
	defer fp.Close()
	err = t.Execute(fp, dReq)
	if err != nil {
		log.WithFields(log.Fields{
			"source":  STORLOGPRE,
			"error":   err,
			"target":  target,
			"request": dReq,
		}).Error("Could not execute the update index template")
		return err
	}
	// todo error
	fp.Chmod(0777)

	return err
}
