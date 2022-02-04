package main

import (
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
