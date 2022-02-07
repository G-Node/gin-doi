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

func TestMKhtml(t *testing.T) {
	// setup temp directory
	targetpath, err := ioutil.TempDir("", "test_cli_html")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(targetpath)

	clioption := "make-html"
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

	// create local test XML file server
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

	// test safe base file creation on non-xml datacite content
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

	// test safe bsae file creation on invalid url
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
	if len(fi) != 1 {
		t.Fatalf("Encountered unexpected number of files: %d/0", len(fi))
	}

	// test valid xml file handling
	testXML := fmt.Sprintf("%s/xml", server.URL)
	cmd.SetArgs([]string{clioption, fmt.Sprintf("-o%s", targetpath), testXML, testXML})
	err = cmd.Execute()
	if err != nil {
		t.Fatalf("Error on valid file URL: %s", err.Error())
	}

	// check valid output directories and file
	target := filepath.Join(targetpath, "10.12751", "g-node.noex1st", "index.html")
	_, err = os.Stat(target)
	if errors.Is(err, os.ErrNotExist) {
		t.Fatalf("Missing output file: %q", target)
	} else if err != nil {
		t.Fatalf("Error accessing file: %s", err.Error())
	}
}
