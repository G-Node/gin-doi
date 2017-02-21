package ginDoi

import (
	"os"
	"fmt"
	"log"
	"html/template"
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

func (ls *LocalStorage) tar(target string) error {
	to := fmt.Sprintf("%s%s", ls.Path, target)
	log.Printf("Will work on:%s", to)
	fp,err := os.Create(to+"/data.tar.gz")
	err = Tar(to+"/tmp", fp )

	return err
}

func (ls *LocalStorage) prepDir(target string, info DoiInfo) error {
	log.Printf("Trying to create:%s",fmt.Sprintf("%s%s", ls.Path, target))
	err := os.Mkdir(fmt.Sprintf("%s/%s", ls.Path, target), os.ModePerm)
	if err != nil{
		log.Print(err)
		return err
	}
	// Deny access per default
	file, err := os.Create(target+"/.httaccess")
	if err != nil{
		log.Printf("Tried httaccess:%s", err)
	}
	file.Write([]byte("deny from all"))

	tmpl, err := template.ParseFiles("tmpl/doiInfo.html")
	if err != nil{
		log.Printf("Trying building a template went wrong:%s", err)
		return err
	}
	fp, _ :=os.Create(target+"/index.html")
	tmpl.Execute(fp, info)
	log.Printf("Got doifile:%+v",info)
	return nil
}

func (ls *LocalStorage) Put(source string , target string) error{
	info, err := ls.Source.GetDoiInfo(source)
	if err != nil{
		log.Printf("Error when fetching doiInfo:%s", err)
	}
	ls.prepDir(target, info)
	ds,_ := ls.GetDataSource()
	to := fmt.Sprintf("%s%s", ls.Path, target)
	if out, err := ds.Get(source, to+"/tmp"); err!=nil {
		return fmt.Errorf("Git said:%s, Error was: %v", out, err)
	}
	err = ls.tar(target)
	err = os.RemoveAll(to+"/tmp")
	ls.DProvider.RegDoi(target)
	return  err
}

func (ls LocalStorage) GetDataSource() (*GinDataSource, error) {
	return &ls.Source, nil
}