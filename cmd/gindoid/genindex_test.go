package main

import (
	"sort"
	"testing"
)

func TestDoilist(t *testing.T) {
	titlefirst := "Pickman's Model"
	titlesecond := "The Doom that Came to Sarnath"
	titlethird := "The Statement of Randolph Carter"

	// test sort by date descending
	dois := []doiitem{
		{
			Title:   titlethird,
			Isodate: "1919-12-03",
		},
		{
			Title:   titlefirst,
			Isodate: "1926-09-01",
		},
	}

	if dois[0].Title != titlethird {
		t.Fatalf("Fail setting up doilist, wrong item order: %v", dois)
	}

	sort.Sort(doilist(dois))
	if dois[0].Title != titlefirst {
		t.Fatalf("Failed sorting by Isodate: %v", dois)
	}

	// test secondary sort by title when dates are identical
	dois = append(dois,
		doiitem{
			Title:   titlesecond,
			Isodate: "1919-12-03",
		})

	sort.Sort(doilist(dois))
	if dois[1].Title != titlesecond {
		t.Fatalf("Failed secondary sorting by Title: %v", dois)
	}
}
