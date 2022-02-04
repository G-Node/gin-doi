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

func TestGetGINDataciteURL(t *testing.T) {
	// test error on empty input
	_, err := getGINDataciteURL("")
	if err == nil {
		t.Fatal("Expected fail on empty input")
	}

	// test error on missing separator
	_, err = getGINDataciteURL("unsupported entry")
	if err == nil {
		t.Fatal("Expected fail on non parsable input")
	}

	// test valid input
	_, err = getGINDataciteURL("repoown/reponame")
	if err != nil {
		t.Fatalf("Unexpected error: %s", err.Error())
	}
}

func TestMKxml(t *testing.T) {
	// setup temp directory
	targetpath, err := ioutil.TempDir("", "test_cli_xml")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(targetpath)

	clioption := "make-xml"
	cmd := setUpCommands("")

	// check safe exit on non-existing output directory
	cmd.SetArgs([]string{clioption, "-oidonotexist", "non-existing.yml"})
	err = cmd.Execute()
	if err != nil {
		t.Fatalf("Error on non-existing output directory: %s", err.Error())
	}

	// check safe exit, no file created on non-existing input file
	cmd.SetArgs([]string{clioption, fmt.Sprintf("-o%s", targetpath), "non-existing.yml"})
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

	// test safe exit, no file created on invalid file url, non-existing gin repo
	// this pings GIN server actual; could be refactored to avoid this fact.
	testInvalidURL := fmt.Sprintf("%s/not-available", server.URL)
	cmd.SetArgs([]string{clioption, fmt.Sprintf("-o%s", targetpath), testInvalidURL, "GIN:invalid", "GIN:not/exist"})
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

	// test safe exit, no file created on empty and invalid yaml file
	testInvalidYML := fmt.Sprintf("%s/non-xml", server.URL)
	testEmpty := fmt.Sprintf("%s/empty", server.URL)
	cmd.SetArgs([]string{clioption, fmt.Sprintf("-o%s", targetpath), testInvalidYML, testEmpty})
	err = cmd.Execute()
	if err != nil {
		t.Fatalf("Error on invalid file: %s", err.Error())
	}
	fi, err = ioutil.ReadDir(targetpath)
	if err != nil {
		t.Fatalf("Error on reading target dir: %s", err.Error())
	}
	if len(fi) != 0 {
		t.Fatalf("Encountered unexpected number of files: %d/0", len(fi))
	}

	// test valid file creation on valid file and no error on empty file
	testValidYML := fmt.Sprintf("%s/dc-yml", server.URL)
	cmd.SetArgs([]string{clioption, fmt.Sprintf("-o%s", targetpath), testValidYML})
	err = cmd.Execute()
	if err != nil {
		t.Fatalf("Error on invalid file: %s", err.Error())
	}

	// check valid output directory and file
	target := filepath.Join(targetpath, "index-000", "doi.xml")
	_, err = os.Stat(target)
	if errors.Is(err, os.ErrNotExist) {
		t.Fatalf("Missing output file: %q", target)
	} else if err != nil {
		t.Fatalf("Error accessing file: %s", err.Error())
	}
}
