package ginDoi

import (
	"os"
	"fmt"
)

type LocalStorage struct {
	Path string
	Source GinDataSource
}

func (ls *LocalStorage) Exists(target string) (bool, error) {
	return false, nil
}

func (ls *LocalStorage) Put(source string , target string) (string, error){
	os.Mkdir(fmt.Sprintf("%s/%s", ls.Path, target),os.ModeDir)
	os.Chdir(fmt.Sprintf("%s/%s", ls.Path, target))
	ds,_ := ls.GetDataSource()
	out, err :=ds.Get(source)
	return string(out), err
}

func (ls LocalStorage) GetDataSource() (*GinDataSource, error) {
	return &ls.Source, nil
}