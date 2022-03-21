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
}
