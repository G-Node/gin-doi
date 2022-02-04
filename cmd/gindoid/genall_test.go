package main

import (
	"fmt"
	"io/ioutil"
	"net/url"
	"os"
	"testing"
)

func TestMKall(t *testing.T) {
	// setup temp directory
	targetpath, err := ioutil.TempDir("", "test_cli_all")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(targetpath)

	clioption := "make-all"
	cmd := setUpCommands("")

	// only check the CLI option with one valid file,
	// the detailled tests should be handled via testing
	// the subcommands.

	// create local test file server
	server := serveDataciteServer()
	defer server.Close()

	// check local test server works
	serverURL, err := url.Parse(server.URL)
	if err != nil {
		t.Fatalf("Could not parse server URL: %q", serverURL)
	}

	// test no error on valid file
	testValidXML := fmt.Sprintf("%s/xml", server.URL)
	cmd.SetArgs([]string{clioption, fmt.Sprintf("-o%s", targetpath), testValidXML})
	err = cmd.Execute()
	if err != nil {
		t.Fatalf("Error on valid file: %s", err.Error())
	}
	fi, err := ioutil.ReadDir(targetpath)
	if err != nil {
		t.Fatalf("Error on reading target dir: %s", err.Error())
	}
	if len(fi) == 0 {
		t.Fatal("Found no output files on valid input")
	}
}
