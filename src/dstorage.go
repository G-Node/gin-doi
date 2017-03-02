package ginDoi

import (
	"os"
	"fmt"
	"log"
	"html/template"
)

var(
	STORLOGPRE = "Storage"

)
type LocalStorage struct {
	Path string
	Source GinDataSource
	DProvider DoiProvider
	HttpBase string
}

func (ls *LocalStorage) Exists(target string) (bool, error) {
	return false, nil
}

func (ls *LocalStorage) tar(target string) (int64, error) {
	to := fmt.Sprintf("%s%s", ls.Path, target)
	log.Printf("[%s] Will work on:%s",STORLOGPRE, to)
	fp,err := os.Create(to+"/data.tar.gz")
	defer fp.Close()
	err = Tar(to+"/tmp", fp )
	stat, _ := fp.Stat()
	return stat.Size(), err
}

func (ls *LocalStorage) prepDir(target string, info *CBerry) error {
	log.Printf("Trying to create:%s",fmt.Sprintf("%s%s", ls.Path, target))
	err := os.Mkdir(fmt.Sprintf("%s%s", ls.Path, target), os.ModePerm)
	if err != nil{
		log.Print(err)
		return err
	}
	// Deny access per default
	file, err := os.Create(fmt.Sprintf("%s%s", ls.Path, target)+"/.htaccess")
	if err != nil{
		log.Printf("Tried httaccess:%s", err)
		return err
	}
	defer file.Close()
	// todo check
	file.Write([]byte("alow from all"))
	return nil
}

func (ls LocalStorage) createIndexFile(target string, info *DoiReq) error {
	tmpl, err := template.ParseFiles("tmpl/doiInfo.html")
	if err != nil{
		log.Printf("[%s] Trying building the index template went wrong:%s", STORLOGPRE, err)
		return err
	}
	fp, err :=os.Create(fmt.Sprintf("%s%s", ls.Path, target)+"/index.html")
	if err != nil{
		log.Printf("[%s] Tried creating index.html:%s",STORLOGPRE, err)
		return err
	}
	defer fp.Close()
	log.Printf("[%s] Will try to create an index.html with: %+v", STORLOGPRE, info)
	if err := tmpl.Execute(fp, info); err!=nil{
		log.Printf("[s] Could not execute template for index.html: %s", STORLOGPRE, err)
		return err
	}
	return nil
}

func (ls *LocalStorage) Put(source string , target string, dReq *DoiReq) error{
	//todo do this better
	dReq.DoiInfo.UUID = target
	ls.prepDir(target, &dReq.DoiInfo)
	ds,_ := ls.GetDataSource()
	to := fmt.Sprintf("%s%s", ls.Path, target)
	if out, err := ds.Get(source, to+"/tmp"); err!=nil {
		return fmt.Errorf("Git said:%s, Error was: %v", out, err)
	}
	fSize, err := ls.tar(target)
	if err != nil {
		log.Printf("[%s] Could not tar: %s", STORLOGPRE,  err)
		return err
	}
	// +1 to report something with small datsets
	dReq.DoiInfo.FileSize = fSize/1000000+1
	ls.createIndexFile(target, dReq)

	err = os.RemoveAll(to+"/tmp")

	fp,_ := os.Create(to+"/doi.xml")
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
	return err
}

func (ls LocalStorage) GetDataSource() (*GinDataSource, error) {
	return &ls.Source, nil
}