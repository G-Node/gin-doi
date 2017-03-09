package ginDoi

import (
	"os"
	"fmt"
	"log"
	"html/template"
	"path/filepath"
	"io"
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
	log.Printf("[%s] Will work on:%s",STORLOGPRE, to)
	fp,err := os.Create(filepath.Join(to, "data.tar.gz"))
	defer fp.Close()
	err = Tar(filepath.Join(to, tmpdir), fp)
	stat, _ := fp.Stat()
	return stat.Size(), err
}

func (ls *LocalStorage) prepDir(target string, info *CBerry) error {
	log.Printf("[%s] Trying to create: %s",STORLOGPRE, filepath.Join(ls.Path, target))
	err := os.Mkdir(filepath.Join(ls.Path, target), os.ModePerm)
	if err != nil{
		log.Printf("[%s] Tried to create the target directory. Smth went wrong: %+v", STORLOGPRE, err)
		return err
	}
	// Deny access per default
	file, err := os.Create(filepath.Join(ls.Path, target,".htaccess"))
	if err != nil{
		log.Printf("[%s]Tried httaccess:%s", STORLOGPRE, err)
		return err
	}
	defer file.Close()
	// todo check
	file.Write([]byte("alow from all"))
	return nil
}

func (ls LocalStorage) createIndexFile(target string, info *DoiReq) error {
	tmpl, err := template.ParseFiles(filepath.Join("tmpl", "doiInfo.html"))
	if err != nil{
		log.Printf("[%s] Trying building the index template went wrong:%s", STORLOGPRE, err)
		return err
	}

	fp, err :=os.Create(filepath.Join(ls.Path, target, "index.html"))
	if err != nil{
		log.Printf("[%s] Tried creating index.html:%s", STORLOGPRE, err)
		return err
	}
	defer fp.Close()
	log.Printf("[%s] Will try to create an index.html with: %+v", STORLOGPRE, info)
	if err := tmpl.Execute(fp, info); err!=nil{
		log.Printf("[%s] Could not execute template for index.html: %s", STORLOGPRE, err)
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
		log.Printf("[%s] Could not tar: %s", STORLOGPRE,  err)
		return err
	}
	// +1 to report something with small datsets
	dReq.DoiInfo.FileSize = fSize/(1024*1000)+1
	ls.createIndexFile(target, dReq)

	err = os.RemoveAll(tmpDir)

	fp,_ := os.Create(filepath.Join(to, "doi.xml"))
	if err != nil{
		log.Printf("[%s] could not create metadata file:%s", STORLOGPRE, err)
		return err
	}
	defer fp.Close()
	// No registering. But the xml is provided with everything
	data, err := ls.DProvider.GetXml(&dReq.DoiInfo)
	if err != nil{
		log.Printf("[%s] could not create metadata: %s", STORLOGPRE, err)
		return err
	}
	_, err = fp.Write(data)
	if err != nil{
		log.Printf("[%s] could not write to metadata file: %s", STORLOGPRE, err)
		return err
	}
	ls.Poerl(to)
	ls.SendMaster(dReq)
	return err
}

func (ls LocalStorage) GetDataSource() (*GinDataSource, error) {
	return &ls.Source, nil
}

func (ls LocalStorage) SendMaster(dReq *DoiReq) (error) {
	return ls.MServer.ToMaster(
		fmt.Sprintf(
			"Hello. the fellowing Archives are ready for doification:%s",
			dReq.DoiInfo.UUID))
}

func (ls LocalStorage) Poerl(to string) (error) {
	pScriptF, err := os.Open(filepath.Join("script","mds-suite_test.pl"))
	if err != nil{
		log.Printf("[%s] The ugly Perls script is not there. Fuck it: %s", STORLOGPRE, err)
		return err
	}
	defer pScriptF.Close()

	pScriptT, err := os.Create(filepath.Join(to,"resgister.pl"))
	if err != nil{
		log.Printf("[%s] The ugly Perls script cannot be created. Screw it: %s", STORLOGPRE, err)
		return err
	}
	defer pScriptT.Close()

	_, err = io.Copy(pScriptT, pScriptF)
	if err != nil{
		log.Printf("[%s] The ugly Perls script cannot be written. HATE IT: %s", STORLOGPRE, err)
		return err
	}

	pScriptT.Chmod(0777)

	return err
}