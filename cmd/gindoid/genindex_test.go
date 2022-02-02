package main

import (
	"errors"
	"fmt"
	"io/ioutil"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"testing"
)

func TestDoilist(t *testing.T) {
	titlefirst := "Pickman's Model"
	titlesecond := "The Doom that Came to Sarnath"
	titlethird := "The Statement of Randolph Carter"

	// test sort by date descending
	dois := []doiitem{
		{
			Title:   titlethird,
			Isodate: "1919-12-03",
		},
		{
			Title:   titlefirst,
			Isodate: "1926-09-01",
		},
	}

	if dois[0].Title != titlethird {
		t.Fatalf("Fail setting up doilist, wrong item order: %v", dois)
	}

	sort.Sort(doilist(dois))
	if dois[0].Title != titlefirst {
		t.Fatalf("Failed sorting by Isodate: %v", dois)
	}

	// test secondary sort by title when dates are identical
	dois = append(dois,
		doiitem{
			Title:   titlesecond,
			Isodate: "1919-12-03",
		})

	sort.Sort(doilist(dois))
	if dois[1].Title != titlesecond {
		t.Fatalf("Failed secondary sorting by Title: %v", dois)
	}
}

func TestMKIndex(t *testing.T) {
	// setup temp directory
	targetpath, err := ioutil.TempDir("", "test_cli_index")
	if err != nil {
		t.Fatalf("temp dir creation failed: %s", err.Error())
	}
	defer os.RemoveAll(targetpath)

	targetFile := filepath.Join(targetpath, "index.html")
	clioption := "make-index"
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
	server := serveDataciteXMLserver()
	defer server.Close()

	// check local test server works
	serverURL, err := url.Parse(server.URL)
	if err != nil {
		t.Fatalf("Could not parse server URL: %q", serverURL)
	}

	// test save exit, no file created on invalid url
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

	// test save exit, no file created on invalid url
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

	// test save exit, no file created on non-xml datacite content
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

	_, err = os.Stat(targetFile)
	if errors.Is(err, os.ErrNotExist) {
		t.Fatalf("Missing output file: %s", targetFile)
	} else if err != nil {
		t.Fatalf("Error writing output file: %s", err.Error())
	}
}
