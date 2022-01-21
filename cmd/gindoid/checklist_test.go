package main

import (
	"testing"

	"github.com/G-Node/libgin/libgin"
)

func TestChecklistFromMetadata(t *testing.T) {
	// assert no issue on blank input
	md := new(libgin.RepositoryMetadata)
	_, err := checklistFromMetadata(md, "")
	if err == nil {
		t.Fatalf("Expected function to fail gracefully")
	}
}
