package ginDoi

import (
	"testing"
	"reflect"
)



func TestDoiFile(t *testing.T) {
	var data = `
title: test
uri: test
authors: testme

`
	mockstruc := DoiInfo{Title:"test",URI:"test",Authors:"testme"}
	val, info := validDoiFile([]byte(data))
	if ! val && reflect.DeepEqual(info,mockstruc){
		t.Logf("Did not properly unmarshjal Doi date:%+v", info)
		t.Fail()
	}
	t.Log("[OK] Unmarschaling doi files works")
}

