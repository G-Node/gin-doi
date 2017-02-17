package ginDoi

import (
	"os/exec"
	"strings"
	"fmt"
	"net/http"
	"io/ioutil"
	"gopkg.in/yaml.v2"
	"log"
)

type GinDataSource struct {
	GinURL string
}

// Return true if the specifies URI "has" a doi File containing all nec. information
func validDoiFile(in []byte) (bool, DoiInfo) {
	//Workaround as long as repo does spit out object type and size
	in =[]byte(strings.Split(string(in),"blob")[0])
	doiInfo := DoiInfo{}
	fmt.Println(string(in))
	err :=yaml.Unmarshal(in, &doiInfo)
	//ToDo check fields and catch error
	if err!=nil{
		fmt.Println(err)
		return false, doiInfo
	}
	return true, doiInfo
}


func (s *GinDataSource) GetDoiFile(URI string) ([]byte, error){
	//git archive --remote=git://git.foo.com/project.git HEAD:path/to/directory filename 
	//https://github.com/go-yaml/yaml.git
	//git@github.com:go-yaml/yaml.git
	fetchRepoPath := ""
	log.Printf("GinDatSource: Got URI:%s", URI)
	if splUri:=strings.Split(URI, "/");len(splUri)>1 {
		uname := strings.Split(splUri[0],":")[1]
		repo := splUri[1]
		fetchRepoPath = fmt.Sprintf("/users/%s/repos/%s/browse/master/doifile.yaml",uname, repo)
	} else {
		return nil,nil 
	}
	log.Printf("GinDatSource: Fetching Path: %s", fetchRepoPath)
	resp, err  := http.Get(fmt.Sprintf("%s%s",s.GinURL, fetchRepoPath))
	if err != nil{
		// todo Try to infer what went wrong
		return nil, err
	}
	
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err !=nil{
		return nil, err
	}
	return body, nil
}

func (s *GinDataSource)  Get(URI string, To string) (string, error) {
	log.Printf("GinDataSource: Will do a git clone now: %s to:%s", URI, To)
	cmd := exec.Command("git","clone","--depth","1", URI, To)
	out, err :=cmd.CombinedOutput()
	if (err != nil) {
		return string(out), err
	}
	return string(out), nil
}