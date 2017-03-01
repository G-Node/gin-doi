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
	log.Printf("Will work on:%s", to)
	fp,err := os.Create(to+"/data.tar.gz")
	defer fp.Close()
	err = Tar(to+"/tmp", fp )
	stat, _ := fp.Stat()
	return stat.Size(), err
}

func (ls *LocalStorage) prepDir(target string, info DoiInfo) error {
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

func (ls LocalStorage) createIndexFile(target string, info DoiInfo) error {
	tmpl, err := template.ParseFiles("tmpl/doiInfo.html")
	if err != nil{
		log.Printf("Trying building a template went wrong:%s", err)
		return err
	}
	fp, err :=os.Create(fmt.Sprintf("%s%s", ls.Path, target)+"/index.html")
	if err != nil{
		log.Printf("Tried creating index.html:%s", err)
		return err
	}
	defer fp.Close()
	if err := tmpl.Execute(fp, info); err!=nil{
		log.Printf("Could not execte template for index.html: %s",err)
		return err
	}
	return nil
}

func (ls *LocalStorage) Put(source string , target string) error{
	info, err := ls.Source.GetDoiInfo(source)
	//todo do this better
	info.UUID = target
	if err != nil{
		log.Printf("Error when fetching doiInfo:%s", err)
	}
	ls.prepDir(target, info)
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

	info.FileSize = fSize/1000
	ls.createIndexFile(target, info)

	err = os.RemoveAll(to+"/tmp")

	fp,_ := os.Create(to+"/doi.xml")
	if err != nil{
		log.Printf("[%s] could not create metadata file:%s", STORLOGPRE, err)
		return err
	}
	defer fp.Close()
	// No registering. But the xml is provided with everything
	data, err := ls.DProvider.GetXml(info)
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