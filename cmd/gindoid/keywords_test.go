package main

import (
	"errors"
	"fmt"
	"io/ioutil"
	"net/url"
	"os"
	"path/filepath"
	"testing"
)

func TestMKkeywords(t *testing.T) {
	// setup temp directory
	targetpath, err := ioutil.TempDir("", "test_cli_keywords")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(targetpath)

	clioption := "make-keyword-pages"
	cmd := setUpCommands("")

	// check safe exit on non-existing output directory
	cmd.SetArgs([]string{clioption, "-oidonotexist", "non-existing.xml"})
	err = cmd.Execute()
	if err != nil {
		t.Fatalf("Error on non-existing output directory: %s", err.Error())
	}

	// check safe exit, no file created on non-existing input file
	cmd.SetArgs([]string{clioption, fmt.Sprintf("-o%s", targetpath), "non-existing.xml"})
	err = cmd.Execute()
	if err != nil {
		t.Fatalf("Error on non-existing input file: %s", err.Error())
	}
	fi, err := ioutil.ReadDir(targetpath)
	if err != nil {
		t.Fatalf("Error on reading target dir: %s", err.Error())
	}
	if len(fi) != 0 {
		t.Fatalf("Encountered unexpected number of files: %d/0", len(fi))
	}

	// create local test file server
	server := serveDataciteServer()
	defer server.Close()

	// check local test server works
	serverURL, err := url.Parse(server.URL)
	if err != nil {
		t.Fatalf("Could not parse server URL: %q", serverURL)
	}

	// test safe exit, no file created on invalid url
	testInvalidURL := fmt.Sprintf("%s/not-available", server.URL)
	cmd.SetArgs([]string{clioption, fmt.Sprintf("-o%s", targetpath), testInvalidURL})
	err = cmd.Execute()
	if err != nil {
		t.Fatalf("Error on invalid file URL: %s", err.Error())
	}
	fi, err = ioutil.ReadDir(targetpath)
	if err != nil {
		t.Fatalf("Error on reading target dir: %s", err.Error())
	}
	if len(fi) != 0 {
		t.Fatalf("Encountered unexpected number of files: %d/0", len(fi))
	}

	// test safe exit, no file created on invalid url
	testEmptyXML := fmt.Sprintf("%s/empty-xml", server.URL)
	cmd.SetArgs([]string{clioption, fmt.Sprintf("-o%s", targetpath), testEmptyXML})
	err = cmd.Execute()
	if err != nil {
		t.Fatalf("Error on empty XML file URL: %s", err.Error())
	}
	fi, err = ioutil.ReadDir(targetpath)
	if err != nil {
		t.Fatalf("Error on reading target dir: %s", err.Error())
	}
	if len(fi) != 0 {
		t.Fatalf("Encountered unexpected number of files: %d/0", len(fi))
	}

	// test safe exit, no file created on non-xml datacite content
	testNonXML := fmt.Sprintf("%s/non-xml", server.URL)
	cmd.SetArgs([]string{clioption, fmt.Sprintf("-o%s", targetpath), testNonXML})
	err = cmd.Execute()
	if err != nil {
		t.Fatalf("Error on non-xml file URL: %s", err.Error())
	}
	fi, err = ioutil.ReadDir(targetpath)
	if err != nil {
		t.Fatalf("Error on reading target dir: %s", err.Error())
	}
	if len(fi) != 0 {
		t.Fatalf("Encountered unexpected number of files: %d/0", len(fi))
	}

	// test valid xml file handling
	testXML := fmt.Sprintf("%s/xml", server.URL)
	cmd.SetArgs([]string{clioption, fmt.Sprintf("-o%s", targetpath), testXML, testXML})
	err = cmd.Execute()
	if err != nil {
		t.Fatalf("Error on valid file URL: %s", err.Error())
	}

	// check keyword list index page
	targetListIndex := filepath.Join(targetpath, "keywords", "index.html")
	_, err = os.Stat(targetListIndex)
	if errors.Is(err, os.ErrNotExist) {
		t.Fatalf("Missing output file: %s", targetListIndex)
	} else if err != nil {
		t.Fatalf("Error accessing file: %s", err.Error())
	}

	// check keyword landing index page
	targetKeyIndex := filepath.Join(targetpath, "keywords", "test_keyword", "index.html")
	_, err = os.Stat(targetKeyIndex)
	if errors.Is(err, os.ErrNotExist) {
		t.Fatalf("Missing output file: %s", targetKeyIndex)
	} else if err != nil {
		t.Fatalf("Error accessing file: %s", err.Error())
	}

	targetDir := filepath.Join(targetpath, "keywords")
	fi, err = ioutil.ReadDir(targetDir)
	if err != nil {
		t.Fatalf("Error on reading target dir: %s", err.Error())
	}
	if len(fi) != 2 {
		t.Fatalf("Encountered unexpected number of files: %d/0", len(fi))
	}
}
