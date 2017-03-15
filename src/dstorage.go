package ginDoi

import (
	"os"
	"fmt"
	log "github.com/Sirupsen/logrus"
	"html/template"
	"path/filepath"
	"io"
	txtTemplate "text/template"
)

var(
	STORLOGPRE = "Storage"
	tmpdir = "tmp"
)

type LocalStorage struct {
	Path string
	Source GinDataSource
	DProvider DoiProvider
	HttpBase string
	MServer *MailServer
}

func (ls *LocalStorage) Exists(target string) (bool, error) {
	return false, nil
}

func (ls *LocalStorage) tar(target string) (int64, error) {
	to := filepath.Join(ls.Path, target)
	log.WithFields(log.Fields{
		"source": STORLOGPRE,
		"to": to,
	}).Debug("Started taring")
	fp,err := os.Create(filepath.Join(to, "data.tar.gz"))
	defer fp.Close()
	err = Tar(filepath.Join(to, tmpdir), fp)
	stat, _ := fp.Stat()
	return stat.Size(), err
}

func (ls *LocalStorage) prepDir(target string, info *CBerry) error {
	err := os.Mkdir(filepath.Join(ls.Path, target), os.ModePerm)
	if err != nil{
		log.WithFields(log.Fields{
			"source": STORLOGPRE,
			"error": err,
			"target": target,
		}).Error("Could not create the target directory")
		return err
	}
	// Deny access per default
	file, err := os.Create(filepath.Join(ls.Path, target,".htaccess"))
	if err != nil{
		log.WithFields(log.Fields{
			"source": STORLOGPRE,
			"error": err,
			"target": target,
		}).Error("Could not create .httaccess")
		return err
	}
	defer file.Close()
	// todo check
	_, err = file.Write([]byte("deny from all"))
	if err != nil{
		log.WithFields(log.Fields{
			"source": STORLOGPRE,
			"error": err,
			"target": target,
		}).Error("Could not write to .httaccess")
		return err
	}
	return nil
}

func (ls LocalStorage) createIndexFile(target string, info *DoiReq) error {
	tmpl, err := template.ParseFiles(filepath.Join("tmpl", "doiInfo.html"))
	if err != nil{
		if err != nil{
			log.WithFields(log.Fields{
				"source": STORLOGPRE,
				"error": err,
				"target": target,
			}).Error("Could not parse the doi template")
			return err
		}
		return err
	}

	fp, err :=os.Create(filepath.Join(ls.Path, target, "index.html"))
	if err != nil{
		log.WithFields(log.Fields{
			"source": STORLOGPRE,
			"error": err,
			"target": target,
		}).Error("Could not create the doi index.html")
		return err
	}
	defer fp.Close()
	if err := tmpl.Execute(fp, info); err!=nil{
		log.WithFields(log.Fields{
			"source": STORLOGPRE,
			"error": err,
			"doiInfoo": info,
		}).Error("Could not execute the doi template")
		return err
	}
	return nil
}

func (ls *LocalStorage) Put(source string , target string, dReq *DoiReq) error{
	//todo do this better
	to := filepath.Join(ls.Path, target)
	tmpDir := filepath.Join(to, tmpdir)
	dReq.DoiInfo.UUID = target
	ls.prepDir(target, &dReq.DoiInfo)
	ds,_ := ls.GetDataSource()

	if out, err := ds.Get(source, tmpDir); err!=nil {
		return fmt.Errorf("[%s] Git said:%s, Error was: %v",STORLOGPRE, out, err)
	}
	fSize, err := ls.tar(target)
	if err != nil {
		log.WithFields(log.Fields{
			"source": STORLOGPRE,
			"error": err,
			"target": target,
		}).Error("Could not tar the data")
		return err
	}
	// +1 to report something with small datsets
	dReq.DoiInfo.FileSize = fSize/(1024*1000)+1
	ls.createIndexFile(target, dReq)

	err = os.RemoveAll(tmpDir)

	fp,_ := os.Create(filepath.Join(to, "doi.xml"))
	if err != nil{
		log.WithFields(log.Fields{
			"source": STORLOGPRE,
			"error": err,
			"target": target,
		}).Error("Could not create parse the metadata template")
		return err
	}
	defer fp.Close()
	// No registering. But the xml is provided with everything
	data, err := ls.DProvider.GetXml(&dReq.DoiInfo)
	if err != nil{
		log.WithFields(log.Fields{
			"source": STORLOGPRE,
			"error": err,
			"target": target,
		}).Error("Could not create the metadata file")
		return err
	}
	_, err = fp.Write(data)
	if err != nil{
		log.WithFields(log.Fields{
			"source": STORLOGPRE,
			"error": err,
			"target": target,
		}).Error("Could not write to the metadata file")
		return err
	}
	ls.Poerl(to)
	ls.MKUpdIndexScript(to, dReq)
	ls.SendMaster(dReq)
	return err
}

func (ls LocalStorage) GetDataSource() (*GinDataSource, error) {
	return &ls.Source, nil
}

func (ls LocalStorage) SendMaster(dReq *DoiReq) (error) {
	return ls.MServer.ToMaster(
		fmt.Sprintf(
			"Hello. the fellowing Archives are ready for doification:%s. Creator:%s",
			dReq.DoiInfo.UUID, string(dReq.User.MainOId.EmailRaw)))
}

func (ls LocalStorage) Poerl(target string) (error) {
	pScriptF, err := os.Open(filepath.Join("script","mds-suite_test.pl"))
	if err != nil{
		log.WithFields(log.Fields{
			"source": STORLOGPRE,
			"error":  err,
			"target": target,
		}).Debug("The ugly Perls script is not there. Fuck it")
		return err
	}
	defer pScriptF.Close()

	pScriptT, err := os.Create(filepath.Join(target,"resgister.pl"))
	if err != nil{
		log.WithFields(log.Fields{
			"source": STORLOGPRE,
			"error":  err,
			"target": target,
		}).Debug("The ugly Perls script cannot be created. Screw it")
		return err
	}
	defer pScriptT.Close()

	_, err = io.Copy(pScriptT, pScriptF)
	if err != nil{
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

func (ls LocalStorage) MKUpdIndexScript(target string, dReq *DoiReq) (error) {
	t, err := txtTemplate.ParseFiles(filepath.Join("tmpl", "updIndex.sh"))
	if err != nil{
		log.WithFields(log.Fields{
			"source": STORLOGPRE,
			"error": err,
			"target": target,
		}).Error("Could not create parse the update index template")
		return err
	}
	fp,_ := os.Create(filepath.Join(target, "updIndex.sh"))
	if err != nil{
		log.WithFields(log.Fields{
			"source": STORLOGPRE,
			"error": err,
			"target": target,
		}).Error("Could not create update index script")
		return err
	}
	defer fp.Close()
	err = t.Execute(fp,dReq)
	if err != nil{
		log.WithFields(log.Fields{
			"source": STORLOGPRE,
			"error": err,
			"target": target,
			"request": dReq,
		}).Error("Could not execute the update index template")
		return err
	}
	// todo error
	fp.Chmod(0777)

	return err
}