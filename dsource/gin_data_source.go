package dsource

import (
	"os/exec"
)

type GinDataSource struct {
	uuid string
}


func (s *GinDataSource) ValidDoiFile(URI string) (bool, error){
	return true, nil
}

func (s *GinDataSource)  Get(URI string) (string, error) {
	cmd := exec.Command("git","clone","--depth","1", URI)
	out, err :=cmd.CombinedOutput()
	if (err != nil) {
		return string(out), err
	}
	return string(out), nil
}