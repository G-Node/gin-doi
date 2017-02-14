package ginDoi

import (
	"os/exec"
	"strings"
	"fmt"
	"net/http"
	"io/ioutil"
	"gopkg.in/yaml.v2"
)

type GinDataSource struct {
	uuid string	
}

// Return true if the specifies URI "has" a doi File containing all nec. information
func validDoiFile(in []byte) (bool, DoiInfo) {
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


func (s *GinDataSource) GetDoiFile(URI *string) ([]byte, error){
	//git archive --remote=git://git.foo.com/project.git HEAD:path/to/directory filename 
	//https://github.com/go-yaml/yaml.git
	//git@github.com:go-yaml/yaml.git
	fetchRepoPath := ""
	if splUri:=strings.Split(*URI, "/");len(splUri)>1 {
		uname := strings.Split(splUri[0],":")[1]
		repo := splUri[1]
		fetchRepoPath = fmt.Sprintf("/users/%s/repos/%s/browse/{branch}/doifile.yaml",uname, repo)
	} else {
		return nil,nil 
	}
	resp, err  := http.Get(fmt.Sprintf("https://repo.gin.g-node.org%s", fetchRepoPath))
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

func (s *GinDataSource)  Get(URI string) (string, error) {
	cmd := exec.Command("git","clone","--depth","1", URI)
	out, err :=cmd.CombinedOutput()
	if (err != nil) {
		return string(out), err
	}
	return string(out), nil
}