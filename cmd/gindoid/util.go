package main

import (
	"encoding/json"
	"fmt"
	log "github.com/Sirupsen/logrus"
	"html/template"
	"io/ioutil"
	"net/http"
	"path/filepath"
	"crypto/aes"
	"crypto/rand"
	"io"
	"crypto/cipher"
	"encoding/base64"
	"time"
)

// Check the current user. Return a user if logged in
func loggedInUser(r *http.Request, pr *OauthProvider) (*DoiUser, error) {
	return &DoiUser{}, nil
}

func readBody(r *http.Request) (*string, error) {
	body, err := ioutil.ReadAll(r.Body)
	x := string(body)
	return &x, err
}

// encrypt string to base64 crypto using AES
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

// decrypt from base64 to decrypted string
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

func IsRegsitredDoi(doi string) (bool) {
	url := fmt.Sprintf("https://doi.org/%s", doi)
	resp, err := http.Get(url)
	if err != nil {
		log.Errorf("Could not querry for doi:%d at %s", doi, url)
		return false
	}
	if resp.StatusCode != http.StatusNotFound {
		return true
	}
	return false
}

func DoDoiJob(w http.ResponseWriter, r *http.Request, jobQueue chan DoiJob, storage LocalStorage, op OauthProvider) {
	// Make sure we can only be called with an HTTP POST request.
	if r.Method != "POST" {
		w.Header().Set("Allow", "POST")
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	dReq := DoiReq{}
	//ToDo Error checking
	body, _ := ioutil.ReadAll(r.Body)
	json.Unmarshal(body, &dReq)
	log.WithFields(log.Fields{
		"request": fmt.Sprintf("%+v", dReq),
		"source":  "DoDoiJob",
	}).Debug("Unmarshaled a doi request")

	ok, err := op.ValidateToken(dReq.OauthLogin, dReq.Token)
	if err != nil {
		log.WithFields(log.Fields{
			"request": fmt.Sprintf("%+v", dReq),
			"source":  "DoDoiJob",
			"error":   err,
		}).Debug("User authentication Failed")
		dReq.Mess = template.HTML(MS_NOLOGIN)
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	if ! ok {
		log.WithFields(log.Fields{
			"request": fmt.Sprintf("%+v", dReq),
			"source":  "DoDoiJob",
		}).Debug("Token not valid")
		dReq.Mess = template.HTML(MS_NOLOGIN)
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	user, err := op.getUser(dReq.OauthLogin, dReq.Token)
	if err != nil {
		log.WithFields(log.Fields{
			"request": fmt.Sprintf("%+v", dReq),
			"source":  "DoDoiJob",
			"error":   err,
		}).Debug("Could not get userdata")
		dReq.Mess = template.HTML(MS_NOLOGIN)
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	dReq.User = DoiUser{MainOId: user}
	//ToDo Error checking
	ds, _ := storage.GetDataSource()
	uuid, _ := ds.MakeUUID(dReq.URI, user)
	if ok, doiInfo := ds.ValidDoiFile(dReq.URI, user); !ok {
		w.WriteHeader(http.StatusBadRequest)
		return
	} else {
		doiInfo.UUID = uuid
		doi := storage.DProvider.MakeDoi(doiInfo)
		dReq.DoiInfo = doiInfo
		key, err := op.AuthorizePull(user)
		if err != nil {
			log.WithFields(log.Fields{
				"source": "DoDoiJob",
				"error":  err,
			}).Error("Could not Authorize Pull")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		if IsRegsitredDoi(doi) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(fmt.Sprintf(MS_DOIREG, doi, doi)))
			return
		}
		job := DoiJob{Source: dReq.URI, Storage: storage, User: user, DoiReq: dReq, Name: doiInfo.UUID, Key: *key}
		jobQueue <- job
		// Render success.
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(fmt.Sprintf(MS_SERVERWORKS, doi, doi)))
	}
}

func InitDoiJob(w http.ResponseWriter, r *http.Request, ds DataSource, op OauthProvider,
	tp string, storage *LocalStorage, key string) {
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
			"source": "InitDoiJob",
			"error":  err,
		}).Error("Could not decrypt token")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	username := r.Form.Get("user")
	dReq := DoiReq{URI: URI, OauthLogin: username, Token: token}
	log.WithFields(log.Fields{
		"request": fmt.Sprintf("%+v", dReq),
		"source":  "Init",
	}).Debug("Got DOI Request")

	t, err := template.ParseFiles(filepath.Join(tp, "initjob.html")) // Parse template file.
	if err != nil {
		log.WithFields(log.Fields{
			"source": "DoDoiJob",
			"error":  err,
		}).Debug("Could not parse init template")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// Test whether URi was provided
	if !(len(URI) > 0) {
		log.WithFields(log.Fields{
			"request": dReq,
			"source":  "Init",
			"error":   err,
		}).Debug("No Repo URI provided")
		dReq.Mess = template.HTML(MS_URIINVALID)
		err := t.Execute(w, dReq)
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
	if !(len(token) > 0) {
		dReq.Mess = template.HTML(MS_NOTOKEN)
		log.WithFields(log.Fields{
			"request": dReq,
			"source":  "Init",
			"error":   err,
		}).Debug("No Token provided")
		err := t.Execute(w, dReq)
		if err != nil {
			log.Print(err)
			return
		}
		return
	}

	// Test whether username was provided
	if !(len(username) > 0) {
		dReq.Mess = template.HTML(MS_NOUSER)
		err := t.Execute(w, dReq)
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
			"source":  "InitDoiJob",
			"error":   err,
		}).Debug("User authentication Failed")
		dReq.Mess = template.HTML(MS_NOLOGIN)
		w.WriteHeader(http.StatusOK)
		return
	}
	if ! ok {
		log.WithFields(log.Fields{
			"request": fmt.Sprintf("%+v", dReq),
			"source":  "InitDoiJob",
		}).Debug("Token not valid")
		dReq.Mess = template.HTML(MS_NOLOGIN)
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
		dReq.Mess = template.HTML(MS_NOLOGIN)
		t.Execute(w, dReq)
		return
	}

	// check for doifile
	if ok, doiInfo := ds.ValidDoiFile(URI, user); ok {
		log.WithFields(log.Fields{
			"doiInfo": doiInfo,
			"source":  "Init",
		}).Debug("Received Doi information")
		dReq.DoiInfo = doiInfo
		err := t.Execute(w, dReq)
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
		}).Debug("Doifile File invalid")
		if doiInfo.Missing != nil {
			dReq.Mess = template.HTML(MS_INVALIDDOIFILE + " <p>Issue:<i> " + doiInfo.Missing[0]+"</i>")
		} else {
			dReq.Mess = template.HTML(MS_INVALIDDOIFILE + MS_ENCODING)
		}
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
	} else {
		dReq.Mess = template.HTML(MS_INVALIDDOIFILE)
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

type DoiMData struct {
	Data struct {
		ID   string `json:"id"`
		Type string `json:"type"`
		Attributes struct {
			Doi        string      `json:"doi"`
			Identifier string      `json:"identifier"`
			URL        interface{} `json:"url"`
			Author []struct {
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
			Results []struct {
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

//https://api.datacite.org/works/
func GDoiMData(doi, doireg string) (*DoiMData, error) {
	url := fmt.Sprintf("%s%s", doireg, doi)
	resp, err := http.Get(url)
	defer resp.Body.Close()
	if err != nil {
		return nil, err
	}
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
	data := &DoiMData{}
	json.Unmarshal(d, data)
	return data, nil
}
