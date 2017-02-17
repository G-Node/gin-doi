package ginDoi

import (
	"os"
	"fmt"
	"log"
	"os/exec"
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

func (ls *LocalStorage) tar(target string) ([]byte, error) {
	//tar -cf archiv.tar daten/
	//Whether the -C option is a good idea otr actually dangerous has to be seen
	to := fmt.Sprintf("%s%s", ls.Path, target)
	//todo check whether -C really changes directories. Would be dangerous here
	cmd := exec.Command("tar","--remove-files","-C",to+"/tmp","-czf", to+"/all.tar.gz", "./")
	out, err :=cmd.CombinedOutput()
	return out, err
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

func (ls *LocalStorage) Put(source string , target string) (string, error){
	info, err := ls.Source.GetDoiInfo(source)
	if err != nil{
		log.Printf("Error when fetching doiinfo:%s", err)
	}
	ls.prepDir(target, info)
	ds,_ := ls.GetDataSource()
	to := fmt.Sprintf("%s%s", ls.Path, target)
	if out, err := ds.Get(source, to+"/tmp"); err!=nil {
		return string(out), err
	}
	out,err := ls.tar(target)
	return string(out), err
}

func (ls LocalStorage) GetDataSource() (*GinDataSource, error) {
	return &ls.Source, nil
}