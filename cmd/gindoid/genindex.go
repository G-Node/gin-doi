package main

import (
	"encoding/xml"
	"fmt"
	"os"
	"sort"
	"strings"
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
		fmt.Printf("Error parsing date '%s' of item '%s'", d[i].Isodate, d[i].Title)
	}
	jdate, err := time.Parse("2006-01-02", d[j].Isodate)
	if err != nil {
		fmt.Printf("Error parsing date '%s' of item '%s'", d[j].Isodate, d[j].Title)
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
	fmt.Printf("Generating %d pages\n", len(args))

	var curritems []doiitem
	for idx, filearg := range args {
		fmt.Printf("%3d: %s\n", idx, filearg)
		var contents []byte
		var err error
		if isURL(filearg) {
			contents, err = readFileAtURL(filearg)
		} else {
			contents, err = readFileAtPath(filearg)
		}
		if err != nil {
			fmt.Printf("Failed to read file at %q: %s\n", filearg, err.Error())
			continue
		}

		datacite := new(libgin.DataCite)
		err = xml.Unmarshal(contents, datacite)
		if err != nil {
			fmt.Printf("Failed to unmarshal contents of %q: %s\n", filearg, err.Error())
			continue
		}
		metadata := &libgin.RepositoryMetadata{
			DataCite: datacite,
		}

		authors := make([]string, len(metadata.Creators))
		for idx, author := range metadata.Creators {
			namesplit := strings.SplitN(author.Name, ",", 2) // Author names are LastName, FirstName
			if len(namesplit) != 2 {
				// No comma: Bad input, mononym, or empty field.
				// Trim, add continue.
				authors[idx] = strings.TrimSpace(author.Name)
				continue
			}
			// render as LastName Initials, ...
			firstnames := strings.Fields(namesplit[1])
			var initials string
			for _, name := range firstnames {
				initials += string(name[0])
			}
			authors[idx] = fmt.Sprintf("%s %s", strings.TrimSpace(namesplit[0]), initials)
		}

		curr := doiitem{
			Title:     metadata.Titles[0],
			Shorthash: metadata.Identifier.ID,
			Authors:   strings.Join(authors, ", "),
			Isodate:   metadata.Dates[0].Value,
		}
		curritems = append(curritems, curr)
	}

	fname := "index.html"
	tmpl, err := prepareTemplates("IndexPage")
	if err != nil {
		fmt.Printf("Error preparing template: %s", err.Error())
		return
	}

	fp, err := os.Create(fname)
	if err != nil {
		fmt.Printf("Could not create the landing page file: %s", err.Error())
		return
	}
	defer fp.Close()

	// sorting the list of items by 1) date descending and 2) title ascending
	sort.Sort(doilist(curritems))
	if err := tmpl.Execute(fp, curritems); err != nil {
		fmt.Printf("Error rendering the landing page: %s", err.Error())
		return
	}
}
