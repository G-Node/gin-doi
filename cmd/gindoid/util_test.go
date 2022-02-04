package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/G-Node/libgin/libgin"
)

func TestReadFileAtPath(t *testing.T) {
	_, err := readFileAtPath("I/do/not/exist")
	if err == nil {
		t.Fatal("Missing error opening non existent file.")
	}

	tmpDir, err := ioutil.TempDir("", "test_gindoi_licfromfile")
	if err != nil {
		t.Fatalf("Error creating tmp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	tmpfile := filepath.Join(tmpDir, "tmp.json")
	tmpcont := `[{"some": "data"}]`
	err = writeTmpFile(tmpfile, tmpcont)
	if err != nil {
		t.Fatalf("Error creating tmp file: %q", err.Error())
	}

	cont, err := readFileAtPath(tmpfile)
	if err != nil {
		t.Fatalf("Error reading tmp file: %q", err.Error())
	}
	if strings.Compare(tmpcont, string(cont)) != 0 {
		t.Fatalf("Issues reading file content: %q", cont)
	}
}

func TestReadFileAtURL(t *testing.T) {
	_, err := readFileAtURL("https://I/do/not/exist")
	if err == nil {
		t.Fatal("Missing error opening non existent URL.")
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/invalid", func(rw http.ResponseWriter, req *http.Request) {
		rw.WriteHeader(http.StatusNotFound)
		_, err := rw.Write([]byte(`non-OK`))
		if err != nil {
			t.Fatalf("Could not write invalid response: %q", err.Error())
		}
	})
	mux.HandleFunc("/valid", func(rw http.ResponseWriter, req *http.Request) {
		rw.WriteHeader(http.StatusOK)
		_, err := rw.Write([]byte(`OK`))
		if err != nil {
			t.Fatalf("Could not write valid response: %q", err.Error())
		}
	})

	// Start local test server
	server := httptest.NewServer(mux)
	// Close the server when test finishes
	defer server.Close()

	serverURL, err := url.Parse(server.URL)
	if err != nil {
		t.Fatalf("Could not parse server URL: %q", serverURL)
	}
	testURL := fmt.Sprintf("%s/invalid", server.URL)
	_, err = readFileAtURL(testURL)
	if err == nil || !strings.Contains(err.Error(), "non-OK status") {
		t.Fatalf("Missing non-OK status error: '%v'", err)
	}
	testURL = fmt.Sprintf("%s/valid", server.URL)
	body, err := readFileAtURL(testURL)
	if err != nil {
		t.Fatalf("Error opening existing file: %q", err.Error())
	}
	if strings.Compare("OK", string(body)) != 0 {
		t.Fatalf("Issues reading file content: %q", string(body))
	}
}

func TestDeduplicateValues(t *testing.T) {
	// check empty
	check := []string{}
	out := deduplicateValues(check)
	if !reflect.DeepEqual(check, out) {
		t.Fatalf("Slices (empty) are not equal: %v | %v", check, out)
	}

	// check nothing to deduplicate
	check = []string{"a", "b", "c"}
	out = deduplicateValues(check)
	if !reflect.DeepEqual(check, out) {
		t.Fatalf("Slices (no duplicates) are not equal: %v | %v", check, out)
	}

	// check deduplication
	check = []string{"a", "b", "a", "c"}
	expected := []string{"a", "b", "c"}
	out = deduplicateValues(check)
	if !reflect.DeepEqual(expected, out) {
		t.Fatalf("Slices (duplicates) are not equal: %v | %v", expected, out)
	}

	// check no deduplication on different case
	check = []string{"A", "b", "a", "B", "c"}
	out = deduplicateValues(check)
	if !reflect.DeepEqual(check, out) {
		t.Fatalf("Slices (no case duplicates) are not equal: %v | %v", check, out)
	}
}

// TestAwardNumber checks proper AwardNumber split and return in util.AwardNumber.
func TestAwardNumber(t *testing.T) {
	subname := "funder name"
	subnum := "award; number"

	// Test normal split on semi-colon
	instr := fmt.Sprintf("%s; %s", subname, subnum)
	outstr := AwardNumber(instr)
	if outstr != subnum {
		t.Fatalf("AwardNumber 'normal' parse error: (in) '%s' (out) '%s' (expect) '%s'", instr, outstr, subnum)
	}

	// Test fallback comma split on missing semi-colon
	subnum = "award, number"
	instr = fmt.Sprintf("%s, %s", subname, subnum)
	outstr = AwardNumber(instr)
	if outstr != subnum {
		t.Fatalf("AwardNumber 'fallback' parse error: (in) '%s' (out) '%s' (expect) '%s'", instr, outstr, subnum)
	}

	// Test empty non split return
	subnum = "award number"
	instr = fmt.Sprintf("%s%s", subname, subnum)
	outstr = AwardNumber(instr)
	if outstr != "" {
		t.Fatalf("AwardNumber 'no-split' parse error: (in) '%s' (out) '%s' (expect) ''", instr, outstr)
	}

	// Test no issue on empty string
	_ = AwardNumber("")

	// Test proper split on comma with semi-colon and surrounding whitespaces
	subnumissue := " award, num "
	subnumclean := "award, num"
	instr = fmt.Sprintf("%s;%s", subname, subnumissue)
	outstr = AwardNumber(instr)
	if outstr != subnumclean {
		t.Fatalf("AwardNumber 'issues' parse error: (in) '%s' (out) '%s' (expect) '%s'", instr, outstr, subnumclean)
	}
}

// TestFunderName checks proper FunderName split and return in util.FunderName.
func TestFunderName(t *testing.T) {
	subname := "funder name"
	subnum := "award number"

	// Test normal split on semi-colon
	instr := fmt.Sprintf("%s; %s", subname, subnum)
	outstr := FunderName(instr)
	if outstr != subname {
		t.Fatalf("Fundername 'normal' parse error: (in) '%s' (out) '%s' (expect) '%s'", instr, outstr, subname)
	}

	// Test fallback comma split on missing semi-colon
	instr = fmt.Sprintf("%s, %s", subname, subnum)
	outstr = FunderName(instr)
	if outstr != subname {
		t.Fatalf("Fundername 'fallback' parse error: (in) '%s' (out) '%s' (expect) '%s'", instr, outstr, subname)
	}

	// Test non split return
	instr = fmt.Sprintf("%s%s", subname, subnum)
	outstr = FunderName(instr)
	if outstr != instr {
		t.Fatalf("Fundername 'no-split' parse error: (in) '%s' (out) '%s' (expect) '%s'", instr, outstr, instr)
	}

	// Test no issue on empty string
	_ = FunderName("")

	// Test proper split on comma with semi-colon and surrounding whitespaces
	subnameissue := " funder, name "
	subnameclean := "funder, name"
	instr = fmt.Sprintf("%s;%s", subnameissue, subnum)
	outstr = FunderName(instr)
	if outstr != subnameclean {
		t.Fatalf("Fundername 'issues' parse error: (in) '%s' (out) '%s' (expect) '%s'", instr, outstr, subnameclean)
	}
}

// TestIsURL tests proper URL identification via util.isURL.
func TestIsURL(t *testing.T) {
	testURL := "i/am/no/url"
	if isURL(testURL) {
		t.Fatalf("isURL returned true for test string %q", testURL)
	}

	testURL = "https://i/could/be/a/url"
	if !isURL(testURL) {
		t.Fatalf("isURL returned false for test string %q", testURL)
	}
}

func TestRandAlnum(t *testing.T) {
	// check negative numbers don't break the function
	tstr := randAlnum(-1)
	if tstr != "" {
		t.Fatalf("Expected empty string but got: %s", tstr)
	}

	// check return length
	for i := 0; i <= 1000; i++ {
		tstr := randAlnum(i)
		if len(tstr) != i {
			t.Fatalf("Invalid output string length, expexted length %d: %s", i, tstr)
		}
	}

	// ensure results are unique within a reasonable number of runs
	dupcheck := make(map[string]bool)
	for i := 0; i < 100; i++ {
		tstr := randAlnum(i)
		_, found := dupcheck[tstr]
		if found {
			t.Fatalf("Fo un d duplicate string '%s' on run %d", tstr, i)
		}
		dupcheck[tstr] = false
	}
}

func TestFormatAuthorList(t *testing.T) {
	// assert no issue on blank input
	md := new(libgin.RepositoryMetadata)
	authors := FormatAuthorList(md)
	if authors != "" {
		t.Fatalf("Expected empty string but got: %s", authors)
	}
	md.DataCite = &libgin.DataCite{}
	authors = FormatAuthorList(md)
	if authors != "" {
		t.Fatalf("Expected empty string but got: %s", authors)
	}

	// Test single author, no comma, whitespace trim
	md.DataCite.Creators = []libgin.Creator{
		{Name: " NameA "},
	}
	authors = FormatAuthorList(md)
	if authors != "NameA" {
		t.Fatalf("Expected trimmed string 'NameA' but got: %s", authors)
	}

	// Test single author family name, two given names, whitespace trim
	md.DataCite.Creators = []libgin.Creator{
		{Name: " NameA, GivenAA GivenAB  "},
	}
	authors = FormatAuthorList(md)
	if authors != "NameA GG" {
		t.Fatalf("Expected formatted author string 'A GG' but got: '%s'", authors)
	}

	// Test multiple, simple name authors, whitespace trim
	md.DataCite.Creators = []libgin.Creator{
		{Name: " NameA "},
		{Name: " NameB "},
		{Name: " NameC "},
	}
	authors = FormatAuthorList(md)
	if authors != "NameA, NameB, NameC" {
		t.Fatalf("Expected formatted authors string 'NameA, NameB, NameC' but got: %s", authors)
	}
	// Test multiple, complex name authors, whitespace trim
	md.DataCite.Creators = []libgin.Creator{
		{Name: " NameA, GivenAA "},
		{Name: " NameB "},
		{Name: " NameC, GivenCA GivenCB GivenCC "},
	}
	authors = FormatAuthorList(md)
	if authors != "NameA G, NameB, NameC GGG" {
		t.Fatalf("Expected formatted authors string 'NameA G, NameB, NameC GGG' but got: %s", authors)
	}
}

func TestFormatCitation(t *testing.T) {
	// assert no issue on blank input
	md := new(libgin.RepositoryMetadata)
	cit := FormatCitation(md)
	if cit != "" {
		t.Fatalf("Expected empty citation: %q", cit)
	}

	// author format is tested in its own function
	md.DataCite = &libgin.DataCite{
		Year:   1996,
		Titles: []string{"test-title"},
		Identifier: libgin.Identifier{
			ID: "test-id"},
	}
	cit = FormatCitation(md)
	if cit != " (1996) test-title. G-Node. https://doi.org/test-id" {
		t.Fatalf("Expected different citation: %q", cit)
	}
}
