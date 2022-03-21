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

func TestURLexists(t *testing.T) {
	// Start local test server
	server := serveDataciteServer()
	// Close the server when test finishes
	defer server.Close()

	serverURL, err := url.Parse(server.URL)
	if err != nil {
		t.Fatalf("Could not parse server URL: %q", serverURL)
	}

	// test non-existing URL
	uex := URLexists("i/do/not/exist")
	if uex {
		t.Fatal("Expected false on non-existing URL")
	}

	// test invalid URL
	uex = URLexists(fmt.Sprintf("%s/not-there", server.URL))
	if uex {
		t.Fatal("Expected false on invalid URL")
	}

	// test valid URL
	uex = URLexists(fmt.Sprintf("%s/xml", server.URL))
	if !uex {
		t.Fatal("Expected true on valid URL")
	}
}

func TestHasGitModules(t *testing.T) {
	// Start local test server
	server := serveDataciteServer()
	// Close the server when test finishes
	defer server.Close()

	serverURL, err := url.Parse(server.URL)
	if err != nil {
		t.Fatalf("Could not parse server URL: %q", serverURL)
	}

	// test non-existing URL
	uex := HasGitModules("i/do/not", "exist")
	if uex {
		t.Fatal("Expected false on non-existing URL")
	}

	// test invalid URL
	uex = HasGitModules(server.URL, "not/there")
	if uex {
		t.Fatal("Expected false on invalid URL")
	}

	// test valid URL
	uex = HasGitModules(server.URL, "test")
	if !uex {
		t.Fatal("Expected true on valid URL")
	}
}

func TestRemoteGitCMD(t *testing.T) {
	// check annex is available to the test; stop the test otherwise
	hasAnnex, err := annexAvailable()
	if err != nil {
		t.Fatalf("Error checking git annex: %q", err.Error())
	} else if !hasAnnex {
		t.Skipf("Annex is not available, skipping test...\n")
	}

	targetpath := t.TempDir()

	// check running git command from non existing path
	_, _, err = remoteGitCMD("/I/do/no/exist", false, "version")
	if err == nil {
		t.Fatal("expected error on non existing directory")
	} else if !strings.Contains(err.Error(), "") {
		t.Fatalf("expected path not found error but got %q", err.Error())
	}

	// check running git command
	stdout, stderr, err := remoteGitCMD(targetpath, false, "version")
	if err != nil {
		t.Fatalf("%q, %q, %q", err.Error(), stderr, stdout)
	}
	// check running git annex command
	stdout, stderr, err = remoteGitCMD(targetpath, true, "version")
	if err != nil {
		t.Fatalf("%q, %q, %q", err.Error(), stderr, stdout)
	}
}

func TestMissingAnnexContent(t *testing.T) {
	// check annex is available to the test; stop the test otherwise
	hasAnnex, err := annexAvailable()
	if err != nil {
		t.Fatalf("Error checking git annex: %q", err.Error())
	} else if !hasAnnex {
		t.Skipf("Annex is not available, skipping test...\n")
	}

	targetpath := t.TempDir()

	// test non existing directory error
	_, _, err = missingAnnexContent("/home/not/exist")
	if err == nil {
		t.Fatal("non existing directory should return an error")
	}

	// test non git directory error
	ismissing, misslist, err := missingAnnexContent(targetpath)
	if err == nil {
		t.Fatalf("non git directory should return an error\nmissing: %t\n%q", ismissing, misslist)
	}

	// initialize git directory
	stdout, stderr, err := remoteGitCMD(targetpath, false, "init")
	if err != nil {
		t.Fatalf("could not initialize git repo: %q, %q, %q", err.Error(), stdout, stderr)
	}

	// test git non annex dir error
	ismissing, misslist, err = missingAnnexContent(targetpath)
	if err == nil {
		t.Fatalf("non git annex directory should return an error\nmissing: %t\n%q", ismissing, misslist)
	}

	// initialize annex
	stdout, stderr, err = remoteGitCMD(targetpath, true, "init")
	if err != nil {
		t.Fatalf("could not init annex: %q, %q, %q", err.Error(), stdout, stderr)
	}

	// test git annex dir no error
	ismissing, misslist, err = missingAnnexContent(targetpath)
	if err != nil {
		t.Fatalf("git annex directory should not return an error\n%s\n%s\n%t", err.Error(), misslist, ismissing)
	}

	// check no missing annex files status
	// create annex data file
	fname := "datafile.txt"
	fpath := filepath.Join(targetpath, fname)
	err = ioutil.WriteFile(fpath, []byte("some data"), 0777)
	if err != nil {
		t.Fatalf("Error creating annex data file %q", err.Error())
	}
	// add file to the annex
	stdout, stderr, err = remoteGitCMD(targetpath, true, "add", fpath)
	if err != nil {
		t.Fatalf("error on git annex add file\n%s\n%s\n%s", err.Error(), stdout, stderr)
	}
	// uninit annex file so the cleanup can happen but ignore any further issues
	// the temp folder will get cleaned up eventually anyway.
	defer remoteGitCMD(targetpath, true, "uninit", fpath)

	stdout, stderr, err = remoteGitCMD(targetpath, false, "commit", "-m", "'add annex file'")
	if err != nil {
		t.Fatalf("error on git commit file\n%s\n%s\n%s", err.Error(), stdout, stderr)
	}
	// check no missing annex content
	ismissing, misslist, err = missingAnnexContent(targetpath)
	if err != nil {
		t.Fatalf("missing annex content check should not return any issue\n%s\n%s\nmissing %t", err.Error(), misslist, ismissing)
	} else if ismissing || misslist != "" {
		t.Fatalf("unexpected missing content found: %t, %q", ismissing, misslist)
	}

	// drop annex file content; use --force since the file content is in no other annex repo and annex thoughtfully complains
	stdout, stderr, err = remoteGitCMD(targetpath, true, "drop", "--force", fpath)
	if err != nil {
		t.Fatalf("error on git annex drop content\n%s\n%s\n%s", err.Error(), stdout, stderr)
	}

	// check missing annex content
	ismissing, misslist, err = missingAnnexContent(targetpath)
	if err != nil {
		t.Fatalf("missing annex content check should not return any issue\n%s\n%t\n%s", err.Error(), ismissing, misslist)
	} else if !ismissing || misslist == "" {
		t.Fatalf("missing annex content check should return missing files\n%t\n%s\n", ismissing, misslist)
	} else if !strings.Contains(misslist, fname) {
		t.Fatalf("missing annex content did not identify missing content: %t %q", ismissing, misslist)
	}
}

func TestLockedAnnexContent(t *testing.T) {
	// check annex is available to the test; stop the test otherwise
	hasAnnex, err := annexAvailable()
	if err != nil {
		t.Fatalf("Error checking git annex: %q", err.Error())
	} else if !hasAnnex {
		t.Skipf("Annex is not available, skipping test...\n")
	}

	targetpath := t.TempDir()

	// test non existing directory error
	islocked, locklist, err := lockedAnnexContent("/home/not/exist")
	if err == nil {
		t.Fatalf("non existing directory should return an error (locked %t) %q", islocked, locklist)
	} else if islocked || locklist != "" {
		t.Fatalf("unexpected locked files (locked %t) %q", islocked, locklist)
	}

	// test non git directory error
	islocked, locklist, err = lockedAnnexContent(targetpath)
	if err == nil {
		t.Fatalf("non git directory should return an error (locked %t) %q", islocked, locklist)
	} else if islocked || locklist != "" {
		t.Fatalf("unexpected locked files (locked %t) %q", islocked, locklist)
	}

	// initialize git directory
	stdout, stderr, err := remoteGitCMD(targetpath, false, "init")
	if err != nil {
		t.Fatalf("could not initialize git repo: %q, %q, %q", err.Error(), stdout, stderr)
	}

	// test git non annex dir error
	islocked, locklist, err = lockedAnnexContent(targetpath)
	if err == nil {
		t.Fatalf("non git annex directory should return an error (locked %t) %q", islocked, locklist)
	} else if islocked || locklist != "" {
		t.Fatalf("unexpected locked files (locked %t) %q", islocked, locklist)
	}

	// initialize annex
	stdout, stderr, err = remoteGitCMD(targetpath, true, "init")
	if err != nil {
		t.Fatalf("could not init annex: %q, %q, %q", err.Error(), stdout, stderr)
	}

	// test git annex dir no error on empty directory
	islocked, locklist, err = lockedAnnexContent(targetpath)
	if err != nil {
		t.Fatalf("git annex directory should not return an error (locked %t) %s\n%s", islocked, locklist, err.Error())
	} else if islocked || locklist != "" {
		t.Fatalf("unexpected locked files (locked %t) %q", islocked, locklist)
	}

	// check no locked annex files status
	// create annex data file
	fname := "datafile.txt"
	fpath := filepath.Join(targetpath, fname)
	err = ioutil.WriteFile(fpath, []byte("some data"), 0777)
	if err != nil {
		t.Fatalf("Error creating annex data file %q", err.Error())
	}
	// add file to the annex; note that this will also lock the file by annex default
	stdout, stderr, err = remoteGitCMD(targetpath, true, "add", fpath)
	if err != nil {
		t.Fatalf("error on git annex add file\n%s\n%s\n%s", err.Error(), stdout, stderr)
	}
	// uninit annex file so the cleanup can happen but ignore any further issues
	// the temp folder will get cleaned up eventually anyway.
	defer remoteGitCMD(targetpath, true, "uninit", fpath)

	// check no locked annex content
	islocked, locklist, err = lockedAnnexContent(targetpath)
	if err != nil {
		t.Fatalf("locked annex content check should not return any issue (locked %t) %s\n%s", islocked, locklist, err.Error())
	} else if !islocked || locklist == "" {
		t.Fatalf("unexpected unlocked content (locked %t) %q", islocked, locklist)
	} else if !strings.Contains(locklist, fname) {
		t.Fatalf("locked annex content did not identify locked content: %t %q", islocked, locklist)
	}

	// unlock annex file content
	stdout, stderr, err = remoteGitCMD(targetpath, true, "unlock", fpath)
	if err != nil {
		t.Fatalf("error on git annex lock content\n%s\n%s\n%s", err.Error(), stdout, stderr)
	}

	// check unlocked annex content
	islocked, locklist, err = lockedAnnexContent(targetpath)
	if err != nil {
		t.Fatalf("unlocked annex content check should not return any issue (locked %t) %s\n%s", islocked, locklist, err.Error())
	} else if islocked || locklist != "" {
		t.Fatalf("unexpected locked content (locked %t) %q", islocked, locklist)
	}
}

func TestAnnexSize(t *testing.T) {
	// check annex is available to the test; stop the test otherwise
	hasAnnex, err := annexAvailable()
	if err != nil {
		t.Fatalf("Error checking git annex: %q", err.Error())
	} else if !hasAnnex {
		t.Skipf("Annex is not available, skipping test...\n")
	}

	targetpath := t.TempDir()

	// test non existing directory error
	reposize, err := annexSize("/home/not/exist")
	if err == nil {
		t.Fatalf("non existing directory should return an error %q", reposize)
	} else if reposize != "" {
		t.Fatalf("unexpected return value %q", reposize)
	}

	// test non git directory error
	reposize, err = annexSize(targetpath)
	if err == nil {
		t.Fatalf("non git directory should return an error %q", reposize)
	} else if reposize != "" {
		t.Fatalf("unexpected return value %q", reposize)
	}

	// initialize git directory
	stdout, stderr, err := remoteGitCMD(targetpath, false, "init")
	if err != nil {
		t.Fatalf("could not initialize git repo: %q, %q, %q", err.Error(), stdout, stderr)
	}

	// test git non annex dir error
	reposize, err = annexSize(targetpath)
	if err == nil {
		t.Fatalf("non git annex directory should return an error %q", reposize)
	} else if reposize != "" {
		t.Fatalf("unexpected return value %q", reposize)
	}

	// initialize annex
	stdout, stderr, err = remoteGitCMD(targetpath, true, "init")
	if err != nil {
		t.Fatalf("could not init annex: %q, %q, %q", err.Error(), stdout, stderr)
	}

	// test git annex dir no error on empty directory
	reposize, err = annexSize(targetpath)
	if err != nil {
		t.Fatalf("git annex directory should not return an error %q\n%v", reposize, err)
	} else if reposize == "" {
		t.Fatalf("unexpected return value %q", reposize)
	} else if !strings.Contains(reposize, "0 bytes") {
		t.Fatalf("unexpected return value %q", reposize)
	}

	// create annex data file
	fname := "datafile.txt"
	fpath := filepath.Join(targetpath, fname)
	err = ioutil.WriteFile(fpath, []byte("some data"), 0777)
	if err != nil {
		t.Fatalf("Error creating annex data file %q", err.Error())
	}
	// add file to the annex; note that this will also lock the file by annex default
	stdout, stderr, err = remoteGitCMD(targetpath, true, "add", fpath)
	if err != nil {
		t.Fatalf("error on git annex add file\n%s\n%s\n%s", err.Error(), stdout, stderr)
	}
	// uninit annex file so the cleanup can happen but ignore any further issues
	// the temp folder will get cleaned up eventually anyway.
	defer remoteGitCMD(targetpath, true, "uninit", fpath)

	// check reposize
	reposize, err = annexSize(targetpath)
	if err != nil {
		t.Fatalf("unexpected error on annexSize %q %q", err.Error(), reposize)
	} else if reposize == "" {
		t.Fatalf("unexpected return value %q", reposize)
	} else if !strings.Contains(reposize, "9 bytes") {
		t.Fatalf("expected return value '9 bytes' but got %q", reposize)
	}

	// reposize should remain unchanged on unlocking files
	stdout, stderr, err = remoteGitCMD(targetpath, true, "unlock", fpath)
	if err != nil {
		t.Fatalf("error on git annex lock content\n%s\n%s\n%s", err.Error(), stdout, stderr)
	}

	// check unlocked annex content
	reposize, err = annexSize(targetpath)
	if err != nil {
		t.Fatalf("unexpected error on annexSize %q %q", err.Error(), reposize)
	} else if reposize == "" {
		t.Fatalf("unexpected return value %q", reposize)
	} else if !strings.Contains(reposize, "9 bytes") {
		t.Fatalf("expected return value '9 bytes' but got %q", reposize)
	}
}

func TestUnlockAnnexClone(t *testing.T) {
	// check annex is available to the test; stop the test otherwise
	hasAnnex, err := annexAvailable()
	if err != nil {
		t.Fatalf("Error checking git annex: %q", err.Error())
	} else if !hasAnnex {
		t.Skipf("Annex is not available, skipping test...\n")
	}

	// prepare git annex directory
	targetroot := t.TempDir()

	reponame := "annextest"
	sourcepath := filepath.Join(targetroot, reponame)
	err = os.Mkdir(sourcepath, 0755)
	if err != nil {
		t.Fatalf("Could not create dir %q: %q", sourcepath, err.Error())
	}

	// initialize git directory
	stdout, stderr, err := remoteGitCMD(sourcepath, false, "init")
	if err != nil {
		t.Fatalf("could not initialize git repo: %q, %q, %q", err.Error(), stdout, stderr)
	}
	// initialize annex
	stdout, stderr, err = remoteGitCMD(sourcepath, true, "init")
	if err != nil {
		t.Fatalf("could not init annex: %q, %q, %q", err.Error(), stdout, stderr)
	}
	// create annex data file
	fname := "datafile.txt"
	fpath := filepath.Join(sourcepath, fname)
	err = ioutil.WriteFile(fpath, []byte("some data"), 0777)
	if err != nil {
		t.Fatalf("Error creating annex data file %q", err.Error())
	}
	// add file to the annex; note that this will also lock the file by annex default
	stdout, stderr, err = remoteGitCMD(sourcepath, true, "add", fpath)
	if err != nil {
		t.Fatalf("error on git annex add file\n%s\n%s\n%s", err.Error(), stdout, stderr)
	}
	stdout, stderr, err = remoteGitCMD(sourcepath, false, "commit", "-m", "'add annex file'")
	if err != nil {
		t.Fatalf("error on git commit\n%s\n%s\n%s", err.Error(), stdout, stderr)
	}
	// uninit annex file so the cleanup can happen but ignore any further issues
	// the temp folder will get cleaned up eventually anyway.
	defer remoteGitCMD(sourcepath, true, "uninit", fpath)

	// test unlockAnnexClone func
	// check error on missing directory
	_, err = unlockAnnexClone(reponame, targetroot, "/i/do/not/exist")
	if err == nil {
		t.Fatal("expected clone error on missing base dir")
	}

	// check no issue on duplicateAnnex
	_, err = unlockAnnexClone(reponame, targetroot, sourcepath)
	if err != nil {
		t.Fatalf("error on duplicate: %q", err.Error())
	}
	// uninit annex file so the cleanup can happen but ignore any further issues
	// the temp folder will get cleaned up eventually anyway.
	targetpath := filepath.Join(targetroot, fmt.Sprintf("%s_unlocked", reponame))
	defer remoteGitCMD(targetpath, true, "uninit", fpath)
}

func TestAcceptedAnnexSize(t *testing.T) {
	// check empty string
	if acceptedAnnexSize("", 250) {
		t.Fatal("True on empty string")
	}

	// check non-splitable string
	if acceptedAnnexSize("100kilobytes", 250) {
		t.Fatal("True on invalid string")
	}

	// check unsupported 'unit'
	if acceptedAnnexSize("10.4 petabytes", 250) {
		t.Fatal("True on unsupported unit petabytes")
	}

	// check non parseable size with threshold unit gigabytes
	if acceptedAnnexSize("doesnotconverttofloat gigabytes", 250) {
		t.Fatal("True on non-parsable size")
	}

	// check supported units
	if !acceptedAnnexSize("10.4 bytes", 250) {
		t.Fatal("False on bytes")
	}
	if !acceptedAnnexSize("10.4 kilobytes", 250) {
		t.Fatal("False on kilobytes")
	}
	if !acceptedAnnexSize("10.4 megabytes", 250) {
		t.Fatal("False on megabytes")
	}

	// check supported unit and supported size
	if !acceptedAnnexSize("10.4 gigabytes", 250) {
		t.Fatal("False on allowed gigabytes")
	}

	// check supported unit and unsupported size
	if acceptedAnnexSize("250.1 gigabytes", 250) {
		t.Fatal("True on unsupported size")
	}
	if acceptedAnnexSize("1 terabytes", 250) {
		t.Fatal("True on terabyte")
	}
}
