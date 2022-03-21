package main

import (
	"strings"
	"testing"

	"github.com/G-Node/libgin/libgin"
)

func TestNotifyAdminContent(t *testing.T) {
	// if any of these is not properly initialized,
	// the function will panic.
	testjob := &RegistrationJob{}
	testjob.Config = &Configuration{}
	testjob.Metadata = &libgin.RepositoryMetadata{}
	datacite := libgin.NewDataCite()
	testjob.Metadata.DataCite = &datacite
	testjob.Metadata.RequestingUser = &libgin.GINUser{}

	var errlist []string
	var warnlist []string
	var full bool
	var chash string

	subjbase := "New DOI registration request: "
	noissuestxt := "no issues have been found"

	// A) Test 'no issues' subject and body
	body, subj := notifyAdminContent(testjob, errlist, warnlist, full, chash)
	if subj != subjbase {
		t.Fatalf("Unexpected subject: %q", subj)
	}
	if !strings.Contains(body, noissuestxt) {
		t.Fatalf("Unexpected body: %q", body)
	}

	// B) Test errors and warnings subject and body
	// Test 'errors' only subject and body
	errbody := "errors occurred"
	warnbody := "issues were detected"

	erritem := "An error was found"
	errlist = append(errlist, erritem)
	body, subj = notifyAdminContent(testjob, errlist, warnlist, full, chash)
	if subj != subjbase {
		t.Fatalf("Unexpected subject: %q", subj)
	}
	if !strings.Contains(body, erritem) || !strings.Contains(body, errbody){
		t.Fatalf("Error missing in body: %q", body)
	} else if strings.Contains(body, warnbody) {
		t.Fatalf("Unexpected warning in body: %q", body)
	}

	// Test 'warnings' only subject and body
	errlist = []string{}
	warnitem := "An error was found"
	warnlist = append(warnlist, warnitem)
	warnlist = append(warnlist, warnitem)
	body, subj = notifyAdminContent(testjob, errlist, warnlist, full, chash)
	if subj != subjbase {
		t.Fatalf("Unexpected subject: %q", subj)
	}
	if !strings.Contains(body, warnitem) || !strings.Contains(body, warnbody) || !strings.Contains(body, "2.") {
		t.Fatalf("Warning missing in body: %q", body)
	} else if strings.Contains(body, errbody) {
		t.Fatalf("Unexpected error in body: %q", body)
	}

	// Test 'errors' and 'warnings' subject and body
	errlist = append(errlist, erritem)
	body, subj = notifyAdminContent(testjob, errlist, warnlist, full, chash)
	if subj != subjbase {
		t.Fatalf("Unexpected subject: %q", subj)
	}
	if !strings.Contains(body, warnitem) || !strings.Contains(body, warnbody) || !strings.Contains(body, "2.") {
		t.Fatalf("Warning missing in body: %q", body)
	} else if !strings.Contains(body, erritem) || !strings.Contains(body, errbody){
		t.Fatalf("Error missing in body: %q", body)
	}

	// C) Test 'full' notification subject and Body
	errlist = []string{}
	warnlist = []string{}
	full = true

	// test no panic on empty entries
	_, _ = notifyAdminContent(testjob, errlist, warnlist, full, chash)

	// test id in subject
	sourcerepo := "source_repo"
	testjob.Metadata.SourceRepository = sourcerepo
	_, subj = notifyAdminContent(testjob, errlist, warnlist, full, chash)
	if !strings.HasPrefix(subj, subjbase) || !strings.Contains(subj, sourcerepo) {
		t.Fatalf("Unexpected subject: %q", subj)
	}

	// test urljoin
	testjob.Config.Storage.StoreURL = "https://storage_url.org/"
	testjob.Metadata.Identifier.ID = "/job/id"
	body, _ = notifyAdminContent(testjob, errlist, warnlist, full, chash)
	if !strings.Contains(body, "new DOI registration") || !strings.Contains(body, "DOI target URL: https://storage_url.org/job/id") {
		t.Fatalf("Unexpected body: %q", body)
	}
}
