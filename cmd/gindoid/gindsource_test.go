package main

import (
	"io/ioutil"
	"log"
	"os"
	"testing"
)

func TestGet(t *testing.T) {
	tmpLoc, err := ioutil.TempDir("", "gin_testget")
	if err != nil {
		t.Log("[Err] Could nor create tempory directory for cloning")
		t.Fail()
		return
	}
	ds := GogsDataSource{GinGitURL: ""}
	out, err := ds.CloneRepository("master:../contrib/test", tmpLoc, nil, "")
	defer os.RemoveAll(tmpLoc)
	if err != nil {
		t.Log(out)
		t.Fail()
		return
	}
	t.Log("[OK] Data source seems to clone")

	out, err = ds.CloneRepository("../test_data/test21", "", nil, "")
	if err == nil {
		t.Log(out)
		t.Fail()
	}
	t.Log("[OK] Data source seems to break")
	//todo test annex
}

func TestValidDOIFile(t *testing.T) {
	ds := GogsDataSource{GinURL: "https://gin.g-node.org/"}
	ok, cb := ds.ValidDOIFile("G-Node/Info", OAuthIdentity{})
	if !ok {
		log.Printf("[Err] Could not get valid DOI file")
		t.Fail()
		return
	}
	if cb.Authors[0].FirstName == "Max" {
		t.Log("[OK]: Getting DOI file seems fine")
	}
}
