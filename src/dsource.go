package ginDoi

import (
	"os/exec"
	"strings"
	"fmt"
	"net/http"
	"io/ioutil"
	"gopkg.in/yaml.v2"
	"log"
	"crypto/md5"
	"encoding/hex"
)

var (
	MS_NOTITLE = "No Title provided"
	MS_NOAUTHORS = "No Authors provided"
	MS_NODESC = ""
	MS_NOLIC = ""
)

type GinDataSource struct {
	GinURL string
}

func hasValues(s *DoiInfo) bool{
	if s.Title==""{
		s.Missing = append(s.Missing, MS_NOTITLE)
	}
	if len(s.Authors)== 0 {
		s.Missing = append(s.Missing, MS_NOAUTHORS)
	}
	if s.Description == ""{
		//s.Missing = append(s.Missing, MS_NODESC)
	}
	if s.License ==""{
		//s.Missing = append(s.Missing, MS_NOLIC)
	}
	return len(s.Missing)>0
}
// Return true if the specifies URI "has" a doi File containing all nec. information
func validDoiFile(in []byte) (bool, DoiInfo) {
	//Workaround as long as repo does spit out object type and size
	in =[]byte(strings.Split(string(in),"blob")[0])
	doiInfo := DoiInfo{}
	err :=yaml.Unmarshal(in, &doiInfo)
	if err != nil {
		log.Printf("GinDatSource: Could not unmarshal doifile:%v", err)
		return false, DoiInfo{}
	}
	if !hasValues(&doiInfo) {
		return false, doiInfo
	}
	return true, doiInfo
}

func (s *GinDataSource) GetDoiInfo(URI string) (DoiInfo, error){
	data, err := s.GetDoiFile(URI)
	if err!=nil{
		return DoiInfo{}, err
	}
	valid,info := validDoiFile(data)
	if !valid {
		return info, fmt.Errorf("Not all cloudberry info provided")
	}
	return info, nil
}

func (s *GinDataSource) GetDoiFile(URI string) ([]byte, error){
	//git archive --remote=git://git.foo.com/project.git HEAD:path/to/directory filename 
	//https://github.com/go-yaml/yaml.git
	//git@github.com:go-yaml/yaml.git
	fetchRepoPath := ""
	log.Printf("GinDatSourceGetDoiFile: Got URI:%s", URI)
	if splUri:=strings.Split(URI, "/");len(splUri)>1 {
		uname := strings.Split(splUri[0],":")[1]
		repo := splUri[1]
		fetchRepoPath = fmt.Sprintf("/users/%s/repos/%s/browse/master/.cloudberry.yml",uname, repo)
	} else {
		return nil,nil 
	}
	log.Printf("GinDatSourceGetDoiFile: Fetching Path: %s", fetchRepoPath)
	resp, err  := http.Get(fmt.Sprintf("%s%s",s.GinURL, fetchRepoPath))
	if err != nil{
		// todo Try to infer what went wrong
		log.Printf("GinDatSourceGetDoiFile: Could not get cloudberry: %v", err)
		return nil, err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err !=nil{
		log.Printf("GinDatSourceGetDoiFile: Could nort read Clodberry: %v", err)
		return nil, err
	}
	return body, nil
}

func (s *GinDataSource)  Get(URI string, To string) (string, error) {
	log.Printf("GinDataSourceGet: Will do a git clone now: %s to:%s", URI, To)
	cmd := exec.Command("git","clone","--depth","1", URI, To)
	out, err :=cmd.CombinedOutput()
	if (err != nil) {
		return string(out), err
	}
	return string(out), nil
}

func (s *GinDataSource)  MakeUUID(URI string) (string, error) {
	fetchRepoPath := ""
	log.Printf("GinDatSourceMakeUUID: Got URI:%s", URI)
	if splUri:=strings.Split(URI, "/");len(splUri)>1 {
		uname := strings.Split(splUri[0],":")[1]
		repo := splUri[1]
		fetchRepoPath = fmt.Sprintf("/users/%s/repos/%s/browse/master",uname, repo)
	}
	log.Printf("GinDatSourceMakeUUID: Fetching Path: %s", fetchRepoPath)
	resp, err  := http.Get(fmt.Sprintf("%s%s",s.GinURL, fetchRepoPath))
	if err != nil{
		return "", err
	}
	if bd,err :=ioutil.ReadAll(resp.Body); err != nil{
		return "", err
	}else {
		currMd5 :=  md5.Sum(bd)
		return hex.EncodeToString(currMd5[:]), nil
	}
}