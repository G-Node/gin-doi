package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestReadFileAtPath(t *testing.T) {
	_, err := readFileAtPath("I/do/not/exist")
	if err == nil {
		t.Fatal("Missing error opening non existant file.")
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
