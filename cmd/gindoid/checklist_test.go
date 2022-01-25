package main

import (
	"strings"
	"testing"

	"github.com/G-Node/libgin/libgin"
)

func TestChecklistFromMetadata(t *testing.T) {
	// assert no panic on nil input
	_, err := checklistFromMetadata(nil, "")
	if err == nil || !strings.Contains(err.Error(), "encountered libgin.RepositoryMetadata nil pointer") {
		t.Fatalf("Expected function to fail gracefully with nil pointer error: %v", err)
	}

	// assert no panic on blank RepositoryMetadata struct, nil sub-level structs
	md := new(libgin.RepositoryMetadata)
	_, err = checklistFromMetadata(md, "")
	if err == nil || !strings.Contains(err.Error(), "encountered libgin.RepositoryMetadata nil pointer") {
		t.Fatalf("Expected function to fail gracefully with nil pointer error: %v", err)
	}

	// assert no panic on nil/blank combinations
	md.DataCite = new(libgin.DataCite)
	_, err = checklistFromMetadata(md, "")
	if err == nil || !strings.Contains(err.Error(), "encountered libgin.RepositoryMetadata nil pointer") {
		t.Fatalf("Expected function to fail gracefully with nil pointer error: %v", err)
	}

	md.DataCite = nil
	md.YAMLData = new(libgin.RepositoryYAML)
	_, err = checklistFromMetadata(md, "")
	if err == nil || !strings.Contains(err.Error(), "encountered libgin.RepositoryMetadata nil pointer") {
		t.Fatalf("Expected function to fail gracefully with nil pointer error: %v", err)
	}

	md.DataCite = new(libgin.DataCite)
	_, err = checklistFromMetadata(md, "")
	if err == nil || !strings.Contains(err.Error(), "encountered libgin.RequestingUser nil pointer") {
		t.Fatalf("Expected function to fail gracefully with nil pointer error: %v", err)
	}

	// assert no panic on empty SourceRepository
	md.RequestingUser = new(libgin.GINUser)
	_, err = checklistFromMetadata(md, "")
	if err == nil || !strings.Contains(err.Error(), "cannot parse SourceRepository") {
		t.Fatalf("Expected function to fail gracefully on missing SourceRepository: %v", err)
	}

	// assert no panic on blank DataCite Dates
	md.SourceRepository = "a/b"
	_, err = checklistFromMetadata(md, "")
	if err == nil || !strings.Contains(err.Error(), "missing pubication date") {
		t.Fatalf("Expected function to fail gracefully on missing DataCite.Dates: %v", err)
	}

	// assert no panic on blank libgin.User
	md.DataCite.Dates = append(md.DataCite.Dates, libgin.Date{})
	_, err = checklistFromMetadata(md, "")
	if err != nil {
		t.Fatalf("%s", err.Error())
	}
}
