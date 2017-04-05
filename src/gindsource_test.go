package ginDoi

import (
	"os"
	"testing"
	"io/ioutil"
	"log"
)

func TestGet(t *testing.T) {
	tmpLoc, err := ioutil.TempDir("", "gin_testget")
	if err != nil{
		t.Log("[Err] Could nor create tempory directory for cloning")
		t.Fail()
		return
	}
	ds := GinDataSource{GinGitURL:""}
	out, err := ds.Get("master:../contrib/test", tmpLoc)
	defer os.RemoveAll(tmpLoc)
	if err != nil {
		t.Log(out)
		t.Fail()
		return
	}
	t.Log("[OK] Data source seems to clone")

	out, err = ds.Get("../test_data/test21", "")
	if err == nil {
		t.Log(out)
		t.Fail()
	}
	t.Log("[OK] Data source seems to break")
}

func TestGetDoiInfo(t *testing.T) {
	ds := GinDataSource{GinURL:"https://repo.gin.g-node.org/"}
	cb, err := ds.GetDoiInfo("master:testi/test")
	if err != nil{
		log.Printf("[Err] Could nor get Doifile :%+v", err)
	}
	if (cb.Authors[0].FirstName=="Max") {
		t.Log("[Ok]: Getting cloudberry seems fine")
	}
}