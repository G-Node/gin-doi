package ginDoi

import (
	"fmt"
	log "github.com/Sirupsen/logrus"
	"io/ioutil"
	"net/http"
	"os/exec"
	"strings"
	"time"
	"crypto/rsa"
	"os"
	"path/filepath"
	"encoding/pem"
	"crypto/x509"
	"golang.org/x/crypto/ssh"
	"crypto/md5"
	"encoding/hex"
	"bytes"
	"regexp"
	"gopkg.in/yaml.v2"
)


type GinDataSource struct {
	GinURL    string
	GinGitURL string
	pubKey    string
}

func (s *GinDataSource) getDoiFile(URI string, user OauthIdentity) ([]byte, error) {
	//git archive --remote=git://git.foo.com/project.git HEAD:path/to/directory filename
	//https://github.com/go-yaml/yaml.git
	//git@github.com:go-yaml/yaml.git
	fetchRepoPath := ""
	if splUri := strings.Split(URI, "/"); len(splUri) > 1 {
		uname := strings.Split(splUri[0], ":")[1]
		repo := splUri[1]
		fetchRepoPath = fmt.Sprintf(GINREPODOIPATH, uname, repo)
	} else {
		return nil, nil
	}
	client := &http.Client{}
	req, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("%s%s", s.GinURL, fetchRepoPath), nil)
	req.Header.Add("Auhoroisation", user.Token)
	resp, err := client.Do(req)
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

func (s *GinDataSource) Get(URI string, To string, key *rsa.PrivateKey) (string, error) {
	gin_uri := strings.Replace(URI, "master:", s.GinGitURL, 1)
	log.WithFields(log.Fields{
		"URI":     URI,
		"gin_uri": gin_uri,
		"to":      To,
		"source":  DSOURCELOGPREFIX,
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

	cmd := exec.Command("git", "clone", "--depth", "1", gin_uri, To)
	env := os.Environ()
	// If a key was provided we need to use that with nthe ssh
	if key != nil {
		_, priv_path, err := WriteSSHKeyPair(tmpDir, key)
		if err != nil {
			log.WithFields(log.Fields{
				"source": DSOURCELOGPREFIX,
				"error":  err,
			}).Error("SSH key storing failed")
			return "", err
		}
		env = append(env, fmt.Sprintf("GIT_SSH_COMMAND=ssh -i %s", priv_path))
		cmd.Env = env
	}
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
	cmd.Dir = To
	cmd.Env = env
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

func (s *GinDataSource) MakeUUID(URI string, user OauthIdentity) (string, error) {
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

// Return true if the specifies URI "has" a doi File containing all nec. information
func (s *GinDataSource) ValidDoiFile(URI string, user OauthIdentity) (bool, *CBerry) {
	in, err := s.getDoiFile(URI, user)
	if err != nil {
		return false, nil
	}
	//Workaround as long as repo does spits out object type and size (and a zero termination...)
	in = bytes.Replace(in, []byte("\x00"), []byte(""), -1)
	re := regexp.MustCompile(`blob\W\d+`)
	in = re.ReplaceAll(in, []byte(""))

	doiInfo := CBerry{}
	err = yaml.Unmarshal(in, &doiInfo)
	if err != nil {
		log.WithFields(log.Fields{
			"data":   string(in),
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

type CBerry struct {
	Missing     []string
	DOI         string
	UUID        string
	FileSize    int64
	Title       string
	Authors     []Author
	Description string
	Keywords    []string
	References  []Reference
	Funding     []string
	License     *License
}

func (c *CBerry) GetCitation() string {
	var authors string
	for _, auth := range c.Authors {
		authors += fmt.Sprintf("%s %s, ", auth.LastName, auth.FirstName)
	}
	return fmt.Sprintf("%s (%d) %s. G-Node. doi:%s", authors, time.Now().Year(), c.Title, c.DOI)
}

type Author struct {
	FirstName   string
	LastName    string
	Affiliation string
	ID          string
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

func hasValues(s *CBerry) bool {
	if s.Title == "" {
		s.Missing = append(s.Missing, MS_NOTITLE)
	}
	if len(s.Authors) == 0 {
		s.Missing = append(s.Missing, MS_NOAUTHORS)
	} else {
		for _, auth := range s.Authors {
			if auth.LastName == "" || auth.FirstName == "" {
				s.Missing = append(s.Missing, MS_AUTHORWRONG)
			}
		}
	}
	if s.Description == "" {
		s.Missing = append(s.Missing, MS_NODESC)
	}
	if s.License == nil || s.License.Name == "" || s.License.Url == "" {
		s.Missing = append(s.Missing, MS_NOLIC)
	}
	if s.References != nil {
		for _, ref := range s.References {
			if ref.Name == "" || ref.Reftype == "" {
				s.Missing = append(s.Missing, MS_REFERENCEWRONG)
			}
		}
	}
	return len(s.Missing) == 0
}

func WriteSSHKeyPair(path string, PrKey *rsa.PrivateKey) (string, string, error) {
	// generate and write private key as PEM
	priv_path := filepath.Join(path, "id_rsa")
	pub_path := filepath.Join(path, "id_rsa.pub")
	privateKeyFile, err := os.Create(priv_path)
	defer privateKeyFile.Close()
	if err != nil {
		return "", "", err
	}
	privateKeyPEM := &pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(PrKey)}
	if err := pem.Encode(privateKeyFile, privateKeyPEM); err != nil {
		return "", "", err
	}

	// generate and write public key
	pub, err := ssh.NewPublicKey(&PrKey.PublicKey)
	if err != nil {
		return "", "", err
	}
	err = ioutil.WriteFile(pub_path, ssh.MarshalAuthorizedKey(pub), 0655)
	if err != nil {
		return "", "", err
	}

	return pub_path, priv_path, nil
}
