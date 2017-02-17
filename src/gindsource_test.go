package ginDoi

import (
	"testing"
	"os"
)



func TestGet(t *testing.T) {
	ds := GinDataSource{}
	os.Mkdir("test1", os.ModePerm)
	out, err :=ds.Get("../test_data/test1", "test1")
	defer os.RemoveAll("test1")
	if err!=nil {
		t.Log(out)
		t.Fail()
	}
	t.Log("[OK] Data source seems to clone")

	out, err =ds.Get("../test_data/test21","")
	if err==nil {
		t.Log(out)
		t.Fail()
	}
	t.Log("[OK] Data source seems to break")
}