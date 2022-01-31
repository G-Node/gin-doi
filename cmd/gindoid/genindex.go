package main

import (
	"encoding/xml"
	"log"
	"os"
	"sort"
	"time"

	"github.com/G-Node/libgin/libgin"
	"github.com/spf13/cobra"
)

// doiitem provides basic DOI information required to
// render the root index page.
type doiitem struct {
	Title     string
	Authors   string
	Isodate   string
	Shorthash string
}

// doilist is an implementation of the sort interface to
// sort a list of doiitems descending by date and ascending
// by title.
type doilist []doiitem

func (d doilist) Len() int {
	return len(d)
}

// Less of the doilist implementation should provide the means
// to sort a list of doiitems first by Isodate in descending
// and in case of identical dates by Title in ascending order.
func (d doilist) Less(i, j int) bool {
	idate, err := time.Parse("2006-01-02", d[i].Isodate)
	if err != nil {
		log.Printf("Error parsing date '%s' of item '%s'", d[i].Isodate, d[i].Title)
	}
	jdate, err := time.Parse("2006-01-02", d[j].Isodate)
	if err != nil {
		log.Printf("Error parsing date '%s' of item '%s'", d[j].Isodate, d[j].Title)
	}
	if idate.Equal(jdate) {
		return d[i].Title < d[j].Title
	}

	return idate.After(jdate)
}

func (d doilist) Swap(i, j int) {
	d[i], d[j] = d[j], d[i]
}

// mkindex reads the provided XML files or URLs and generates the HTML landing
// page for each.
func mkindex(cmd *cobra.Command, args []string) {
	log.Printf("Parsing %d files\n", len(args))

	var dois []doiitem
	for idx, filearg := range args {
		log.Printf("%3d: %s\n", idx, filearg)
		var contents []byte
		var err error
		if isURL(filearg) {
			contents, err = readFileAtURL(filearg)
		} else {
			contents, err = readFileAtPath(filearg)
		}
		if err != nil {
			log.Printf("Failed to read file at %q: %s\n", filearg, err.Error())
			continue
		}

		datacite := new(libgin.DataCite)
		err = xml.Unmarshal(contents, datacite)
		if err != nil {
			log.Printf("Failed to unmarshal contents of %q: %s\n", filearg, err.Error())
			continue
		}
		metadata := &libgin.RepositoryMetadata{
			DataCite: datacite,
		}

		curr := doiitem{
			Title:     metadata.Titles[0],
			Shorthash: metadata.Identifier.ID,
			Authors:   FormatAuthorList(metadata),
			Isodate:   metadata.Dates[0].Value,
		}
		dois = append(dois, curr)
	}

	if len(dois) < 1 {
		log.Printf("No DOIs parsed, skipping empty 'index' file creation.")
		return
	}

	fname := "index.html"
	tmpl, err := prepareTemplates("IndexPage")
	if err != nil {
		log.Printf("Error preparing template: %s", err.Error())
		return
	}

	fp, err := os.Create(fname)
	if err != nil {
		log.Printf("Could not create the landing page file: %s", err.Error())
		return
	}
	defer fp.Close()

	// sorting the list of items by 1) date descending and 2) title ascending
	sort.Sort(doilist(dois))
	if err := tmpl.Execute(fp, dois); err != nil {
		log.Printf("Error rendering the landing page: %s", err.Error())
		return
	}
}
