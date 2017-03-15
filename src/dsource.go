package ginDoi

import (
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	log "github.com/Sirupsen/logrus"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"net/http"
	"os/exec"
	"regexp"
	"strings"
)

var (
	MS_NOTITLE       = "No Title provided."
	MS_NOAUTHORS     = "No Authors provided."
	MS_NODESC        = "No Description provided."
	MS_NOLIC         = "No Valid Liecense provided.Plaese specify url and name!"
	MS_REFERENCEWRONG= "A specified Reference is not valid (needs name and type)"
	DSOURCELOGPREFIX = "DataSource"
)

type CBerry struct {
	Missing     []string
	DOI         string
	UUID        string
	FileSize    int64
	Title       string
	Authors     []string
	Description string
	Keywords    []string
	References  []Reference
	Funding     []string
	License     *License
}

type Reference struct {
	Reftype string
	Name    string
	Doi     string
}

type License struct {
	Name string
	Url  string
}

type GinDataSource struct {
	GinURL string
}

func hasValues(s *CBerry) bool {
	if s.Title == "" {
		s.Missing = append(s.Missing, MS_NOTITLE)
	}
	if len(s.Authors) == 0 {
		s.Missing = append(s.Missing, MS_NOAUTHORS)
	}
	if s.Description == "" {
		s.Missing = append(s.Missing, MS_NODESC)
	}
	if s.License == nil || s.License.Name=="" || s.License.Url=="" {
		s.Missing = append(s.Missing, MS_NOLIC)
	}
	if s.References != nil{
		for _, ref := range s.References{
			if ref.Name=="" || ref.Reftype==""{
				s.Missing = append(s.Missing,MS_REFERENCEWRONG)
			}
		}
	}
	return len(s.Missing) == 0
}

// Return true if the specifies URI "has" a doi File containing all nec. information
func validDoiFile(in []byte) (bool, *CBerry) {
	//Workaround as long as repo does spits out object type and size (and a zero termination...)
	in = bytes.Replace(in, []byte("\x00"), []byte(""), -1)
	re := regexp.MustCompile(`blob\W\d+`)
	in = re.ReplaceAll(in, []byte(""))

	doiInfo := CBerry{}
	err := yaml.Unmarshal(in, &doiInfo)
	if err != nil {
		log.WithFields(log.Fields{
			"data":   in,
			"source": DSOURCELOGPREFIX,
			"error":  err,
		}).Error("Could not unmarshal doifile")
		return false, &CBerry{}
	}
	if !hasValues(&doiInfo) {
		log.WithFields(log.Fields{
			"data":    in,
			"doiInfo": doiInfo,
			"source":  DSOURCELOGPREFIX,
			"error":   err,
		}).Debug("Doi File misses entries")
		return false, &doiInfo
	}
	return true, &doiInfo
}

func (s *GinDataSource) GetDoiInfo(URI string) (*CBerry, error) {
	data, err := s.GetDoiFile(URI)
	if err != nil {
		return nil, err
	}
	valid, info := validDoiFile(data)
	if !valid {
		return info, fmt.Errorf("Not all cloudberry info provided")
	}
	return info, nil
}

func (s *GinDataSource) GetDoiFile(URI string) ([]byte, error) {
	//git archive --remote=git://git.foo.com/project.git HEAD:path/to/directory filename
	//https://github.com/go-yaml/yaml.git
	//git@github.com:go-yaml/yaml.git
	fetchRepoPath := ""
	if splUri := strings.Split(URI, "/"); len(splUri) > 1 {
		uname := strings.Split(splUri[0], ":")[1]
		repo := splUri[1]
		fetchRepoPath = fmt.Sprintf("/users/%s/repos/%s/browse/master/cloudberry.yml", uname, repo)
	} else {
		return nil, nil
	}
	resp, err := http.Get(fmt.Sprintf("%s%s", s.GinURL, fetchRepoPath))
	if err != nil {
		// todo Try to infer what went wrong
		log.WithFields(log.Fields{
			"path":   fetchRepoPath,
			"source": DSOURCELOGPREFIX,
			"error":  err,
		}).Debug("Could not get cloudberry")
		return nil, err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.WithFields(log.Fields{
			"path":   fetchRepoPath,
			"source": DSOURCELOGPREFIX,
			"error":  err,
		}).Debug("Could nort read from received Clodberry")
		return nil, err
	}
	return body, nil
}

func (s *GinDataSource) Get(URI string, To string) (string, error) {
	gin_uri := strings.Replace(URI, "master:", "git@gin.g-node.org:", 1)
	log.WithFields(log.Fields{
		"URI":     URI,
		"gin_uri": gin_uri,
		"to":      To,
		"source":  DSOURCELOGPREFIX,
	}).Debug("Start cloning")
	cmd := exec.Command("git", "clone", "--depth", "1", gin_uri, To)
	out, err := cmd.CombinedOutput()
	log.WithFields(log.Fields{
		"URI":     URI,
		"gin_uri": gin_uri,
		"to":      To,
		"out":     string(out),
		"source":  DSOURCELOGPREFIX,
	}).Debug("Done with cloning")
	if err != nil {
		log.WithFields(log.Fields{
			"URI":     URI,
			"gin_uri": gin_uri,
			"to":      To,
			"source":  DSOURCELOGPREFIX,
			"error":   string(out),
		}).Debug("Cloning did not work")
		return string(out), err
	}
	cmd = exec.Command("git", "annex", "sync", "--no-push", "--content")
	out, err = cmd.CombinedOutput()
	if err != nil {
		// Workaround for uninitilaizes git annexes (-> return nil)
		// todo
		log.WithFields(log.Fields{
			"URI":     URI,
			"gin_uri": gin_uri,
			"to":      To,
			"source":  DSOURCELOGPREFIX,
			"error":   string(out),
		}).Debug("Repo was not annexed")
		return string(out), nil
	}
	return string(out), nil
}

func (s *GinDataSource) MakeUUID(URI string) (string, error) {
	fetchRepoPath := ""
	if splUri := strings.Split(URI, "/"); len(splUri) > 1 {
		uname := strings.Split(splUri[0], ":")[1]
		repo := splUri[1]
		fetchRepoPath = fmt.Sprintf("/users/%s/repos/%s/browse/master", uname, repo)
	}
	resp, err := http.Get(fmt.Sprintf("%s%s", s.GinURL, fetchRepoPath))
	// todo error checking
	if err != nil {
		return "", err
	}
	if bd, err := ioutil.ReadAll(resp.Body); err != nil {
		return "", err
	} else {
		currMd5 := md5.Sum(bd)
		return hex.EncodeToString(currMd5[:]), nil
	}
}
