package ginDoi

import (
	"os"
	"fmt"
	"log"
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

func (ls *LocalStorage) Put(source string , target string) (string, error){
	log.Printf("Trying to create:%s",fmt.Sprintf("%s%s", ls.Path, target))
	err := os.Mkdir(fmt.Sprintf("%s/%s", ls.Path, target),os.ModePerm)
	if err != nil{
		log.Print(err)
	}
	ds,_ := ls.GetDataSource()
	out, err :=ds.Get(source, fmt.Sprintf("%s/%s", ls.Path, target))
	return string(out), err
}

func (ls LocalStorage) GetDataSource() (*GinDataSource, error) {
	return &ls.Source, nil
}