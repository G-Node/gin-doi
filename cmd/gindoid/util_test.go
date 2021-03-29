package main

import (
	"fmt"
	"reflect"
	"testing"
)

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
