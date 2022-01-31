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

func TestURLlist(t *testing.T) {
	titlefirst := "The Doom that Came to Sarnath"
	titlesecond := "The Statement of Randolph Carter"
	titlethird := "Pickman's Model"

	// test sort by date ascending
	dois := []doiitem{
		{
			Title:   titlethird,
			Isodate: "1926-09-01",
		},
		{
			Title:   titlefirst,
			Isodate: "1919-12-03",
		},
	}

	if dois[0].Title != titlethird {
		t.Fatalf("Fail setting up doilist, wrong item order: %v", dois)
	}

	sort.Sort(urllist(dois))
	if dois[0].Title != titlefirst {
		t.Fatalf("Failed sorting by Isodate: %v", dois)
	}

	// test secondary sort by title when dates are identical
	dois = append(dois,
		doiitem{
			Title:   titlesecond,
			Isodate: "1919-12-03",
		})

	sort.Sort(urllist(dois))
	if dois[0].Title != titlefirst || dois[1].Title != titlesecond {
		t.Fatalf("Failed secondary sorting by Title: %v", dois)
	}
}

func TestMKSitemap(t *testing.T) {
	targetpath, err := ioutil.TempDir("", "test_sitemap_cli")
	if err != nil {
		t.Fatalf("Failed to create sitemap cli temp directory: %v", err)
	}
	defer os.RemoveAll(targetpath)

	cmd := setUpCommands("")

	// check safe exit on non-existing output directory
	cmd.SetArgs([]string{"make-sitemap", "-oidonotexist", "non-existing.xml"})
	err = cmd.Execute()
	if err != nil {
		t.Fatalf("Error on non-existing sitemap directory: %s", err.Error())
	}

	// check safe exit on non-existing input file
	cmd.SetArgs([]string{"make-sitemap", fmt.Sprintf("-o%s", targetpath), "non-existing.xml"})
	err = cmd.Execute()
	if err != nil {
		t.Fatalf("Error on non-existing sitemap directory: %s", err.Error())
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

	// test error on invalid datacite content
	testNonXML := fmt.Sprintf("%s/non-xml", server.URL)
	cmd.SetArgs([]string{"make-sitemap", fmt.Sprintf("-o%s", targetpath), testNonXML})
	err = cmd.Execute()
	if err != nil {
		t.Fatalf("Error on invalid sitemap file URL: %s", err.Error())
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
	cmd.SetArgs([]string{"make-sitemap", fmt.Sprintf("-o%s", targetpath), testXML, testXML})
	err = cmd.Execute()
	if err != nil {
		t.Fatalf("Error on invalid sitemap file URL: %s", err.Error())
	}
	targetFile := filepath.Join(targetpath, "urls.txt")
	_, err = os.Stat(targetFile)
	if errors.Is(err, os.ErrNotExist) {
		t.Fatalf("Could not find sitemap file at: %s", targetFile)
	} else if err != nil {
		t.Fatalf("Unexpected error writing sitemap file: %s", err.Error())
	}
}
