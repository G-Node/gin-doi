package main

import (
	"crypto/md5"
	"crypto/rsa"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"strings"

	log "github.com/Sirupsen/logrus"
	"github.com/gogits/go-gogs-client"
	"gopkg.in/yaml.v2"
)

type GogsDataSource struct {
	GinURL    string
	GinGitURL string
	pubKey    string
}

func (s *GogsDataSource) getDOIFile(URI string, user OAuthIdentity) ([]byte, error) {
	// git archive --remote=git://git.foo.com/project.git HEAD:path/to/directory filename
	// https://github.com/go-yaml/yaml.git
	// git@github.com:go-yaml/yaml.git
	// TODO: config variables for path etc.
	fetchRepoPath := fmt.Sprintf("%s/raw/master/datacite.yml", URI)
	client := &http.Client{}
	req, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("%s/%s", s.GinURL, fetchRepoPath), nil)
	req.Header.Add("Cookie", fmt.Sprintf("i_like_gogits=%s", user.Token))
	resp, err := client.Do(req)
	if err != nil {
		// todo Try to infer what went wrong
		log.WithFields(log.Fields{
			"path":   fetchRepoPath,
			"source": DSOURCELOGPREFIX,
			"error":  err,
		}).Debug("Could not get DOI file")
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("could not get DOI file: %s", resp.Status)
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.WithFields(log.Fields{
			"path":   fetchRepoPath,
			"source": DSOURCELOGPREFIX,
			"error":  err,
		}).Debug("Could not read from received datacite.yml file")
		return nil, err
	}
	return body, nil
}

func (s *GogsDataSource) Get(URI string, To string, key *rsa.PrivateKey) (string, error) {
	ginURI := fmt.Sprintf("%s/%s.git", s.GinGitURL, strings.ToLower(URI))
	log.WithFields(log.Fields{
		"URI":    URI,
		"ginURI": ginURI,
		"to":     To,
		"source": DSOURCELOGPREFIX,
	}).Debug("Start cloning")

	//Create tmp ssh keys files from the key provided
	tmpDir, err := ioutil.TempDir("", "")
	if err != nil {
		log.WithFields(log.Fields{
			"source": DSOURCELOGPREFIX,
			"error":  err,
		}).Error("SSH key tmp dir not created")
		return "", err
	}

	cmd := exec.Command("git", "clone", ginURI, To)
	env := os.Environ()
	// If a key was provided we need to use that with nthe ssh
	if key != nil {
		_, privPath, err := WriteSSHKeyPair(tmpDir, key)
		if err != nil {
			log.WithFields(log.Fields{
				"source": DSOURCELOGPREFIX,
				"error":  err,
			}).Error("SSH key storing failed")
			return "", err
		}
		env = append(env, fmt.Sprintf("GIT_SSH_COMMAND=ssh -i %s", privPath))
		env = append(env, "GIT_COMMITTER_NAME=GINDOI")
		env = append(env, "GIT_COMMITTER_EMAIL=doi@g-node.org")
		cmd.Env = env
	}
	out, err := cmd.CombinedOutput()
	log.WithFields(log.Fields{
		"URI":     URI,
		"gin_uri": ginURI,
		"to":      To,
		"out":     string(out),
		"source":  DSOURCELOGPREFIX,
	}).Debug("Done with cloning")
	if err != nil {
		log.WithFields(log.Fields{
			"URI":     URI,
			"gin_uri": ginURI,
			"to":      To,
			"source":  DSOURCELOGPREFIX,
			"error":   string(out),
		}).Debug("Cloning did not work")
		return string(out), err
	}

	cmd = exec.Command("git-annex", "get")
	cmd.Dir = To
	cmd.Env = env
	out, err = cmd.CombinedOutput()
	if err != nil {
		// Workaround for uninitilaizes git annexes (-> return nil)
		// todo
		log.WithFields(log.Fields{
			"source": DSOURCELOGPREFIX,
			"error":  string(out),
		}).Debug("Annex get failed")
	}
	cmd = exec.Command("git-annex", "uninit")
	cmd.Dir = To
	cmd.Env = env
	out, err = cmd.CombinedOutput()
	if err != nil {
		// Workaround for uninitilaizes git annexes (-> return nil)
		// todo
		log.WithFields(log.Fields{
			"source": DSOURCELOGPREFIX,
			"error":  string(out),
		}).Debug("Anex unlock failed")
	}
	return string(out), nil
}

var UUIDMap = map[string]string{
	"INT/multielectrode_grasp":                   "f83565d148510fede8a277f660e1a419",
	"ajkumaraswamy/HB-PAC_disinhibitory_network": "1090f803258557299d287c4d44a541b2",
	"steffi/Kleineidam_et_al_2017":               "f53069de4c4921a3cfa8f17d55ef98bb",
	"Churan/Morris_et_al_Frontiers_2016":         "97bc1456d3f4bca2d945357b3ec92029",
	"fabee/efish_locking":                        "6953bbf0087ba444b2d549b759de4a06",
}

func RepoP2UUID(URI string) string {
	if doi, ok := UUIDMap[URI]; ok {
		return doi
	}
	currMd5 := md5.Sum([]byte(URI))
	return hex.EncodeToString(currMd5[:])
}
func (s *GogsDataSource) MakeUUID(URI string, user OAuthIdentity) (string, error) {
	return RepoP2UUID(URI), nil
}

// GetMasterCommit determines the latest commit id of the master branch
func (s *GogsDataSource) GetMasterCommit(URI string, user OAuthIdentity) (string, error) {
	fetchRepoPath := fmt.Sprintf("%s", URI)
	client := &http.Client{}
	req, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("%s/api/v1/repos/%s/branches", s.GinURL, fetchRepoPath), nil)
	req.Header.Add("Cookie", fmt.Sprintf("i_like_gogits=%s", user.Token))
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("Could not get repo branches: %s", resp.Status)
	}

	branches := []gogs.Branch{}
	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	json.Unmarshal(data, &branches)
	for _, branch := range branches {
		if branch.Name == "master" {
			return branch.Commit.ID, nil
		}
	}
	return "", fmt.Errorf("Could not locate master branch")
}

// ValidDOIFile returns true if the specified URI has a DOI file containing all necessary information.
func (s *GogsDataSource) ValidDOIFile(URI string, user OAuthIdentity) (bool, *DOIRegInfo) {
	in, err := s.getDOIFile(URI, user)
	if err != nil {
		log.WithFields(log.Fields{
			"data":   string(in),
			"source": DSOURCELOGPREFIX,
			"error":  err,
		}).Error("Could not get the DOI file")
		return false, nil
	}
	doiInfo := DOIRegInfo{}
	err = yaml.Unmarshal(in, &doiInfo)
	if err != nil {
		log.WithFields(log.Fields{
			"data":   string(in),
			"source": DSOURCELOGPREFIX,
			"error":  err,
		}).Error("Could not unmarshal DOI file")
		res := DOIRegInfo{}
		res.Missing = []string{fmt.Sprintf("%s", err)}
		return false, &res
	}
	if !hasValues(&doiInfo) {
		log.WithFields(log.Fields{
			"data":    in,
			"doiInfo": doiInfo,
			"source":  DSOURCELOGPREFIX,
			"error":   err,
		}).Debug("DOI file misses entries")
		return false, &doiInfo
	}
	return true, &doiInfo
}
