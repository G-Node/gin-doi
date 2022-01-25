package main

import (
	"errors"
	"fmt"
	"io/ioutil"
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
