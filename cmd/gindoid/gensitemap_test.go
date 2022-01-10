package main

import (
	"sort"
	"testing"
)

func TestURLlist(t *testing.T) {
	titlefirst := "The Doom that Came to Sarnath"
	titlesecond := "The Statement of Randolph Carter"
	titlethird := "Pickman's Model"

	// test sort by date ascending
	dois := []urlitem{
		{
			Title:   titlethird,
			Isodate: "1926-09-01",
		},
		{
			Title:   titlefirst,
			Isodate: "1919-12-03",
		},
	}

	if dois[0].Title != titlethird {
		t.Fatalf("Fail setting up doilist, wrong item order: %v", dois)
	}

	sort.Sort(urllist(dois))
	if dois[0].Title != titlefirst {
		t.Fatalf("Failed sorting by Isodate: %v", dois)
	}

	// test secondary sort by title when dates are identical
	dois = append(dois,
		urlitem{
			Title:   titlesecond,
			Isodate: "1919-12-03",
		})

	sort.Sort(urllist(dois))
	if dois[0].Title != titlefirst || dois[1].Title != titlesecond {
		t.Fatalf("Failed secondary sorting by Title: %v", dois)
	}
}
