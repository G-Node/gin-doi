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
	"bytes"
	 "regexp"
)

var (
	MS_NOTITLE = "No Title provided."
	MS_NOAUTHORS = "No Authors provided."
	MS_NODESC = "No Description provided."
	MS_NOLIC = "No Liecense provided."
	DSOURCELOGPREFIX= "DataSource"
)

type GinDataSource struct {
	GinURL string
}

func hasValues(s *CBerry) bool{
	if s.Title == ""{
		s.Missing = append(s.Missing, MS_NOTITLE)
	}
	if len(s.Authors) == 0 {
		s.Missing = append(s.Missing, MS_NOAUTHORS)
	}
	if s.Description == ""{
		s.Missing = append(s.Missing, MS_NODESC)
	}
	if s.License ==""{
		s.Missing = append(s.Missing, MS_NOLIC)
	}
	return len(s.Missing)==0
}
// Return true if the specifies URI "has" a doi File containing all nec. information
func validDoiFile(in []byte) (bool, *CBerry) {
	//Workaround as long as repo does spits out object type and size (and a zero termination...)
	in = bytes.Replace(in,[]byte("\x00"),[]byte(""),-1)
	re := regexp.MustCompile(`blob\W\d+`)
	in = re.ReplaceAll(in, []byte(""))

	doiInfo := CBerry{}
	err :=yaml.Unmarshal(in, &doiInfo)
	if err != nil {
		log.Printf("[%s]: Could not unmarshal doifile:%v",DSOURCELOGPREFIX, err)
		return false, &CBerry{}
	}
	if !hasValues(&doiInfo) {
		log.Printf("[%s]: Doi File misses entries: %+v",DSOURCELOGPREFIX, doiInfo)
		return false, &doiInfo
	}
	return true, &doiInfo
}

func (s *GinDataSource) GetDoiInfo(URI string) (*CBerry, error){
	data, err := s.GetDoiFile(URI)
	if err!=nil{
		return nil, err
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
	log.Printf("[%s]: Got URI:%s", DSOURCELOGPREFIX, URI)
	if splUri:=strings.Split(URI, "/");len(splUri)>1 {
		uname := strings.Split(splUri[0],":")[1]
		repo := splUri[1]
		fetchRepoPath = fmt.Sprintf("/users/%s/repos/%s/browse/master/cloudberry.yml",uname, repo)
	} else {
		return nil,nil 
	}
	log.Printf("[%s]: Fetching Path: %s",DSOURCELOGPREFIX, fetchRepoPath)
	resp, err  := http.Get(fmt.Sprintf("%s%s",s.GinURL, fetchRepoPath))
	if err != nil{
		// todo Try to infer what went wrong
		log.Printf("[%s]: Could not get cloudberry: %v", DSOURCELOGPREFIX, err)
		return nil, err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err !=nil{
		log.Printf("[%s]: Could nort read Clodberry: %v", DSOURCELOGPREFIX, err)
		return nil, err
	}
	return body, nil
}

func (s *GinDataSource)  Get(URI string, To string) (string, error) {
	log.Printf("[%s]: Will do a git clone now: %s to:%s",DSOURCELOGPREFIX, URI, To)
	cmd := exec.Command("git","clone","--depth","1", URI, To)
	out, err := cmd.CombinedOutput()
	if (err != nil) {
		return string(out), err
	}
	cmd = exec.Command("git", "annex", "sync", "--no-push", "--content")
	out, err = cmd.CombinedOutput()
	if (err != nil) {
		// Workaround for uninitilaizes git annexes (-> return nil)
		// todo
		log.Printf("[%s] Repo was not annexed: %s",DSOURCELOGPREFIX,string(out))
		return string(out), nil
	}
	return string(out), nil
}

func (s *GinDataSource) MakeUUID(URI string) (string, error) {
	fetchRepoPath := ""
	log.Printf("[%s]: Got URI for UUIDfication:%s",DSOURCELOGPREFIX, URI)
	if splUri:=strings.Split(URI, "/");len(splUri)>1 {
		uname := strings.Split(splUri[0],":")[1]
		repo := splUri[1]
		fetchRepoPath = fmt.Sprintf("/users/%s/repos/%s/browse/master",uname, repo)
	}
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