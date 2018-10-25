package main

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"html/template"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"time"

	log "github.com/Sirupsen/logrus"
	"golang.org/x/crypto/ssh"
)

// Check the current user. Return a user if logged in
func loggedInUser(r *http.Request, pr *OAuthProvider) (*DOIUser, error) {
	return &DOIUser{}, nil
}

func readBody(r *http.Request) (*string, error) {
	body, err := ioutil.ReadAll(r.Body)
	x := string(body)
	return &x, err
}

// Encrypt string to base64 crypto using AES
func Encrypt(key []byte, text string) (string, error) {
	plaintext := []byte(text)

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}

	// The IV needs to be unique, but not secure. Therefore it's common to
	// include it at the beginning of the ciphertext.
	ciphertext := make([]byte, aes.BlockSize+len(plaintext))
	iv := ciphertext[:aes.BlockSize]
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		return "", err
	}

	stream := cipher.NewCFBEncrypter(block, iv)
	stream.XORKeyStream(ciphertext[aes.BlockSize:], plaintext)

	// convert to base64
	return base64.URLEncoding.EncodeToString(ciphertext), nil
}

// Decrypt from base64 to decrypted string
func Decrypt(key []byte, cryptoText string) (string, error) {
	ciphertext, _ := base64.URLEncoding.DecodeString(cryptoText)

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}

	// The IV needs to be unique, but not secure. Therefore it's common to
	// include it at the beginning of the ciphertext.
	if len(ciphertext) < aes.BlockSize {
		return "", err
	}
	iv := ciphertext[:aes.BlockSize]
	ciphertext = ciphertext[aes.BlockSize:]

	stream := cipher.NewCFBDecrypter(block, iv)

	// XORKeyStream can work in-place if the two arguments are the same.
	stream.XORKeyStream(ciphertext, ciphertext)

	return fmt.Sprintf("%s", ciphertext), nil
}

func IsRegisteredDOI(doi string) bool {
	url := fmt.Sprintf("https://doi.org/%s", doi)
	resp, err := http.Get(url)
	if err != nil {
		log.Errorf("Could not query for doi: %s at %s", doi, url)
		return false
	}
	if resp.StatusCode != http.StatusNotFound {
		return true
	}
	return false
}

func DoDOIJob(w http.ResponseWriter, r *http.Request, jobQueue chan DOIJob, storage LocalStorage, op OAuthProvider) {
	// Make sure we can only be called with an HTTP POST request.
	if r.Method != "POST" {
		w.Header().Set("Allow", "POST")
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	dReq := DOIReq{}
	// TODO: Error checking
	body, _ := ioutil.ReadAll(r.Body)
	json.Unmarshal(body, &dReq)
	log.WithFields(log.Fields{
		"request": fmt.Sprintf("%+v", dReq),
		"source":  "DoDOIJob",
	}).Debug("Unmarshaled a DOI request")

	ok, err := op.ValidateToken(dReq.OAuthLogin, dReq.Token)
	if err != nil {
		log.WithFields(log.Fields{
			"request": fmt.Sprintf("%+v", dReq),
			"source":  "DoDOIJob",
			"error":   err,
		}).Debug("User authentication Failed")
		dReq.Message = template.HTML(MS_NOLOGIN)
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	if !ok {
		log.WithFields(log.Fields{
			"request": fmt.Sprintf("%+v", dReq),
			"source":  "DoDOIJob",
		}).Debug("Token not valid")
		dReq.Message = template.HTML(MS_NOLOGIN)
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	user, err := op.getUser(dReq.OAuthLogin, dReq.Token)
	if err != nil {
		log.WithFields(log.Fields{
			"request": fmt.Sprintf("%+v", dReq),
			"source":  "DoDOIJob",
			"error":   err,
		}).Debug("Could not get userdata")
		dReq.Message = template.HTML(MS_NOLOGIN)
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	dReq.User = DOIUser{MainOId: user}
	// TODO Error checking
	ds := storage.GetDataSource()
	uuid, _ := ds.MakeUUID(dReq.URI, user)
	ok, doiInfo := ds.ValidDOIFile(dReq.URI, user)
	if !ok {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	doiInfo.UUID = uuid
	doi := storage.DProvider.MakeDOI(doiInfo)
	dReq.DOIInfo = doiInfo
	key, err := op.AuthorizePull(user)
	if err != nil {
		log.WithFields(log.Fields{
			"source": "DoDOIJob",
			"error":  err,
		}).Error("Could not Authorize Pull")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if IsRegisteredDOI(doi) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(fmt.Sprintf(MS_DOIREG, doi, doi)))
		return
	}
	job := DOIJob{Source: dReq.URI, Storage: storage, User: user, Request: dReq, Name: doiInfo.UUID, Key: *key}
	jobQueue <- job
	// Render success.
	w.WriteHeader(http.StatusCreated)
	w.Write([]byte(fmt.Sprintf(MS_SERVERWORKS, doi, doi)))
}

func InitDOIJob(w http.ResponseWriter, r *http.Request, ds DataSource, op OAuthProvider, tp string, storage *LocalStorage, key string) {
	log.Infof("Got a new DOI request")
	if err := r.ParseForm(); err != nil {
		log.WithFields(log.Fields{
			"source": "Init",
		}).Debug("Could not parse form data")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	URI := r.Form.Get("repo")
	token := r.Form.Get("token")
	token, err := Decrypt([]byte(key), token)
	if err != nil {
		log.WithFields(log.Fields{
			"source": "InitDOIJob",
			"error":  err,
		}).Error("Could not decrypt token")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	username := r.Form.Get("user")
	dReq := DOIReq{URI: URI, OAuthLogin: username, Token: token}
	log.WithFields(log.Fields{
		"request": fmt.Sprintf("%s (from: %s)", URI, username),
		"source":  "Init",
	}).Debug("Got DOI Request")

	t, err := template.ParseFiles(filepath.Join(tp, "initjob.html")) // Parse template file.
	if err != nil {
		log.WithFields(log.Fields{
			"source": "DoDOIJob",
			"error":  err,
		}).Debug("Could not parse init template")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// Test whether URI was provided
	if len(URI) == 0 {
		log.WithFields(log.Fields{
			"request": dReq,
			"source":  "Init",
			"error":   err,
		}).Debug("No Repo URI provided")
		dReq.Message = template.HTML(MS_URIINVALID)
		err = t.Execute(w, dReq)
		if err != nil {
			log.WithFields(log.Fields{
				"request": dReq,
				"source":  "Init",
				"error":   err,
			}).Debug("Template not parsed")
			return
		}
		return
	}

	// Test whether token was provided
	if len(token) == 0 {
		dReq.Message = template.HTML(MS_NOTOKEN)
		log.WithFields(log.Fields{
			"request": dReq,
			"source":  "Init",
			"error":   err,
		}).Debug("No Token provided")
		err = t.Execute(w, dReq)
		if err != nil {
			log.Print(err)
			return
		}
		return
	}

	// Test whether username was provided
	if len(username) == 0 {
		dReq.Message = template.HTML(MS_NOUSER)
		err = t.Execute(w, dReq)
		if err != nil {
			log.Print(err)
			return
		}
		return
	}

	// test user login
	ok, err := op.ValidateToken(username, token)
	if err != nil {
		log.WithFields(log.Fields{
			"request": fmt.Sprintf("%+v", dReq),
			"source":  "InitDOIJob",
			"error":   err,
		}).Debug("User authentication Failed")
		dReq.Message = template.HTML(MS_NOLOGIN)
		w.WriteHeader(http.StatusOK)
		return
	}
	if !ok {
		log.WithFields(log.Fields{
			"request": fmt.Sprintf("%+v", dReq),
			"source":  "InitDOIJob",
		}).Debug("Token not valid")
		dReq.Message = template.HTML(MS_NOLOGIN)
		w.WriteHeader(http.StatusOK)
		t.Execute(w, dReq)
		return
	}

	// get user
	user, err := op.getUser(username, token)
	if err != nil {
		log.WithFields(log.Fields{
			"request": dReq,
			"source":  "Init",
			"error":   err,
		}).Debug("Could not authenticate user")
		dReq.Message = template.HTML(MS_NOLOGIN)
		t.Execute(w, dReq)
		return
	}

	// check for doifile
	if ok, doiInfo := ds.ValidDOIFile(URI, user); ok {
		log.WithFields(log.Fields{
			"doiInfo": doiInfo,
			"source":  "Init",
		}).Debug("Received DOI information")
		dReq.DOIInfo = doiInfo
		err = t.Execute(w, dReq)
		if err != nil {
			log.WithFields(log.Fields{
				"request": dReq,
				"source":  "Init",
				"error":   err,
			}).Error("Could not parse template")
			return
		}
	} else if doiInfo != nil {
		log.WithFields(log.Fields{
			"doiInfo": doiInfo,
			"source":  "Init",
			"error":   err,
		}).Debug("DOIfile File invalid")
		if doiInfo.Missing != nil {
			dReq.Message = template.HTML(MS_INVALIDDOIFILE + " <p>Issue:<i> " + doiInfo.Missing[0] + "</i>")
		} else {
			dReq.Message = template.HTML(MS_INVALIDDOIFILE + MS_ENCODING)
		}
		dReq.DOIInfo = &DOIRegInfo{}
		err = t.Execute(w, dReq)
		if err != nil {
			log.WithFields(log.Fields{
				"doiInfo": doiInfo,
				"request": dReq,
				"source":  "Init",
				"error":   err,
			}).Error("Could not parse template")
			return
		}
		return
	} else {
		dReq.Message = template.HTML(MS_INVALIDDOIFILE)
		t.Execute(w, dReq)
		if err != nil {
			log.WithFields(log.Fields{
				"request": dReq,
				"source":  "Init",
				"error":   err,
			}).Error("Could not parse template")
			return
		}
		return
	}
}

type DOIMData struct {
	Data struct {
		ID         string `json:"id"`
		Type       string `json:"type"`
		Attributes struct {
			DOI        string      `json:"doi"`
			Identifier string      `json:"identifier"`
			URL        interface{} `json:"url"`
			Author     []struct {
				Literal string `json:"literal"`
			} `json:"author"`
			Title               string      `json:"title"`
			ContainerTitle      string      `json:"container-title"`
			Description         string      `json:"description"`
			ResourceTypeSubtype string      `json:"resource-type-subtype"`
			DataCenterID        string      `json:"data-center-id"`
			MemberID            string      `json:"member-id"`
			ResourceTypeID      string      `json:"resource-type-id"`
			Version             string      `json:"version"`
			License             interface{} `json:"license"`
			SchemaVersion       string      `json:"schema-version"`
			Results             []struct {
				ID    string `json:"id"`
				Title string `json:"title"`
				Count int    `json:"count"`
			} `json:"results"`
			RelatedIdentifiers []struct {
				RelationTypeID    string `json:"relation-type-id"`
				RelatedIdentifier string `json:"related-identifier"`
			} `json:"related-identifiers"`
			Published  string      `json:"published"`
			Registered time.Time   `json:"registered"`
			Updated    time.Time   `json:"updated"`
			Media      interface{} `json:"media"`
			XML        string      `json:"xml"`
		} `json:"attributes"`
		Relationships struct {
			DataCenter struct {
				Meta struct {
				} `json:"meta"`
			} `json:"data-center"`
			Member struct {
				Meta struct {
				} `json:"meta"`
			} `json:"member"`
			ResourceType struct {
				Meta struct {
				} `json:"meta"`
			} `json:"resource-type"`
		} `json:"relationships"`
	} `json:"data"`
}

type DOIinvalid struct {
	error
}

// https://api.datacite.org/works/
func GDOIMData(doi, doireg string) (*DOIMData, error) {
	url := fmt.Sprintf("%s%s", doireg, doi)
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return nil, DOIinvalid{}
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Resource not found")
	}
	d, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	data := &DOIMData{}
	json.Unmarshal(d, data)
	return data, nil
}

func WriteSSHKeyPair(path string, PrKey *rsa.PrivateKey) (string, string, error) {
	// generate and write private key as PEM
	privPath := filepath.Join(path, "id_rsa")
	pubPath := filepath.Join(path, "id_rsa.pub")
	privateKeyFile, err := os.Create(privPath)
	defer privateKeyFile.Close()
	if err != nil {
		return "", "", err
	}
	privateKeyPEM := &pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(PrKey)}
	if err = pem.Encode(privateKeyFile, privateKeyPEM); err != nil {
		return "", "", err
	}
	privateKeyFile.Chmod(0600)
	// generate and write public key
	pub, err := ssh.NewPublicKey(&PrKey.PublicKey)
	if err != nil {
		return "", "", err
	}
	err = ioutil.WriteFile(pubPath, ssh.MarshalAuthorizedKey(pub), 0600)
	if err != nil {
		return "", "", err
	}

	return pubPath, privPath, nil
}
