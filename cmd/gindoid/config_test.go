package main

import (
	"os"
	"strings"
	"testing"
)

func TestLoadconfig(t *testing.T) {
	// check 'configdir' env var
	_, err := loadconfig()
	if err == nil {
		t.Fatal("Expected error on missing 'configdir' env var")
	}

	confpath := t.TempDir()
	err = os.Setenv("configdir", confpath)
	if err != nil {
		t.Fatalf("Error setting 'confdir': %q", err.Error())
	}

	// check 'ginurl' env var
	_, err = loadconfig()
	if err == nil {
		t.Fatal("Expected error on missing 'ginurl' env var")
	} else if err != nil && !strings.Contains(err.Error(), "invalid web configuration") {
		t.Fatalf("Error loading config: %q", err.Error())
	}

	err = os.Setenv("ginurl", "invalidurl")
	if err != nil {
		t.Fatalf("Error setting 'ginurl': %q", err.Error())
	}
	_, err = loadconfig()
	if err == nil {
		t.Fatal("Expected error on invalid 'ginurl' env var")
	} else if err != nil && !strings.Contains(err.Error(), "invalid web configuration") {
		t.Fatalf("Error loading config: %q", err.Error())
	}

	// check invalid giturl
	// we are not testing the gin-cli.config.ParseWebString here
	err = os.Setenv("ginurl", "https://a.valid.url:1221")
	if err != nil {
		t.Fatalf("Error setting 'ginurl': %q", err.Error())
	}
	_, err = loadconfig()
	if err == nil {
		t.Fatal("Expected error on invalid 'giturl' env var")
	} else if err != nil && !strings.Contains(err.Error(), "invalid git configuration") {
		t.Fatalf("Error loading config: %q", err.Error())
	}

	// check error on invalid server setup
	err = os.Setenv("giturl", "git@a.valid.url:2222")
	if err != nil {
		t.Fatalf("Error setting 'giturl': %q", err.Error())
	}
	_, err = loadconfig()
	if err == nil {
		t.Fatal("Expected error on invalid server configuration")
	} else if err != nil && !strings.Contains(err.Error(), "no such host") {
		t.Fatalf("Error loading config: %q", err.Error())
	}

	// further tests require a local git server.
}
