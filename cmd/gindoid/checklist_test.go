package main

import (
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/G-Node/libgin/libgin"
)

func TestChecklistFromMetadata(t *testing.T) {
	// assert no panic on nil input
	_, err := checklistFromMetadata(nil, "")
	if err == nil || !strings.Contains(err.Error(), "encountered libgin.RepositoryMetadata nil pointer") {
		t.Fatalf("Expected function to fail gracefully with nil pointer error: %v", err)
	}

	// assert no panic on blank RepositoryMetadata struct, nil sub-level structs
	md := new(libgin.RepositoryMetadata)
	_, err = checklistFromMetadata(md, "")
	if err == nil || !strings.Contains(err.Error(), "encountered libgin.RepositoryMetadata nil pointer") {
		t.Fatalf("Expected function to fail gracefully with nil pointer error: %v", err)
	}

	// assert no panic on nil/blank combinations
	md.DataCite = new(libgin.DataCite)
	_, err = checklistFromMetadata(md, "")
	if err == nil || !strings.Contains(err.Error(), "encountered libgin.RepositoryMetadata nil pointer") {
		t.Fatalf("Expected function to fail gracefully with nil pointer error: %v", err)
	}

	md.DataCite = nil
	md.YAMLData = new(libgin.RepositoryYAML)
	_, err = checklistFromMetadata(md, "")
	if err == nil || !strings.Contains(err.Error(), "encountered libgin.RepositoryMetadata nil pointer") {
		t.Fatalf("Expected function to fail gracefully with nil pointer error: %v", err)
	}

	md.DataCite = new(libgin.DataCite)
	_, err = checklistFromMetadata(md, "")
	if err == nil || !strings.Contains(err.Error(), "encountered libgin.RequestingUser nil pointer") {
		t.Fatalf("Expected function to fail gracefully with nil pointer error: %v", err)
	}

	// assert no panic on empty SourceRepository
	md.RequestingUser = new(libgin.GINUser)
	_, err = checklistFromMetadata(md, "")
	if err == nil || !strings.Contains(err.Error(), "cannot parse SourceRepository") {
		t.Fatalf("Expected function to fail gracefully on missing SourceRepository: %v", err)
	}

	// assert no panic on blank DataCite Dates
	md.SourceRepository = "a/b"
	_, err = checklistFromMetadata(md, "")
	if err == nil || !strings.Contains(err.Error(), "missing pubication date") {
		t.Fatalf("Expected function to fail gracefully on missing DataCite.Dates: %v", err)
	}

	// assert no panic on blank libgin.User
	md.DataCite.Dates = append(md.DataCite.Dates, libgin.Date{})
	_, err = checklistFromMetadata(md, "")
	if err != nil {
		t.Fatalf("%s", err.Error())
	}
}

func TestWriteReadChecklistConfigYAML(t *testing.T) {
	targetpath, err := ioutil.TempDir("", "test_doi_write_checklist_config")
	if err != nil {
		t.Fatalf("Failed to create checklist config temp directory: %v", err)
	}
	defer os.RemoveAll(targetpath)

	// test no panic on blank input
	cl := checklist{}
	err = writeChecklistConfigYAML(cl, targetpath)
	if err != nil {
		t.Fatalf("Error writing blank checklist config file: %s", err.Error())
	}

	// test writing default config
	cl = defaultChecklist()
	err = writeChecklistConfigYAML(cl, targetpath)
	if err != nil {
		t.Fatalf("Error writing default checklist config file: %s", err.Error())
	}
	targetFile := filepath.Join(targetpath, fmt.Sprintf("conf_%s.yml", cl.Regid))
	_, err = os.Stat(targetFile)
	if errors.Is(err, os.ErrNotExist) {
		t.Fatalf("Could not find checklist config file at: %s", targetFile)
	} else if err != nil {
		t.Fatalf("Unexpected error writing checklist config file: %s", err.Error())
	}

	// test readChecklistConfigYAML
	// test invalid read
	chl := new(checklist)
	_, err = readChecklistConfigYAML(chl, "")
	if err == nil {
		t.Fatalf("Expected read file error")
	} else if err != nil && !strings.Contains(err.Error(), "-- Error reading config file") {
		t.Fatalf("Expected read file error but got: %s", err.Error())
	}

	chl, err = readChecklistConfigYAML(chl, targetFile)
	if err != nil {
		t.Fatalf("Error on reading config file: %s", err.Error())
	}
	compcl := defaultChecklist()
	if !reflect.DeepEqual(compcl, *chl) {
		t.Fatalf("Loaded config differs from original: %v, %v", compcl, *chl)
	}
}

func TestParseRepoDatacite(t *testing.T) {
	dataciteYAML := `title: "title"
authors:
-
 firstname: "firstname"
 lastname: "lastname"
`

	_, _, err := parseRepoDatacite("")
	if err == nil {
		t.Fatal("Missing error on missing URL")
	}
	_, _, err = parseRepoDatacite("https://gin.g-node.org/idonotexist")
	if err == nil {
		t.Fatal("Missing error on invalid URL")
	}

	// provide local test server for datacite yml files
	mux := http.NewServeMux()
	mux.HandleFunc("/non-dc-yml", func(rw http.ResponseWriter, req *http.Request) {
		rw.WriteHeader(http.StatusOK)
		_, err := rw.Write([]byte(`non-yml`))
		if err != nil {
			t.Fatalf("Could not write invalid response: %q", err.Error())
		}
	})
	mux.HandleFunc("/dc-yml", func(rw http.ResponseWriter, req *http.Request) {
		rw.WriteHeader(http.StatusOK)
		_, err := rw.Write([]byte(dataciteYAML))
		if err != nil {
			t.Fatalf("Could not write valid response: %q", err.Error())
		}
	})

	// start local test server
	server := httptest.NewServer(mux)
	defer server.Close()

	// test local test server works
	serverURL, err := url.Parse(server.URL)
	if err != nil {
		t.Fatalf("Could not parse server URL: %q", serverURL)
	}

	// test error on invalid datacite content
	testNonYAML := fmt.Sprintf("%s/non-dc-yml", server.URL)
	_, _, err = parseRepoDatacite(testNonYAML)
	if err == nil || !strings.Contains(err.Error(), "unmarshalling config file") {
		t.Fatalf("Error handling invalid config unmarshal: %v", err)
	}

	// test valid dc yaml file import
	testYAML := fmt.Sprintf("%s/dc-yml", server.URL)
	title, authorlist, err := parseRepoDatacite(testYAML)
	if err != nil {
		t.Fatalf("Error handling valid config unmarshal: %s", err.Error())
	}
	if title != "title" {
		t.Fatalf("Error parsing title from datacite.yml: %s", title)
	}
	if authorlist != "lastname f" {
		t.Fatalf("Error parsing authors from datacite.yml: %s", authorlist)
	}
}
