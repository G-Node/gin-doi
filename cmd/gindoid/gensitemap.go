package main

import (
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"log"
	"path/filepath"
	"sort"
	"time"

	"github.com/G-Node/libgin/libgin"
	"github.com/spf13/cobra"
)

// urllist is an implementation of the sort interface to
// sort a list of doiitems ascending by date and title.
type urllist []doiitem

func (d urllist) Len() int {
	return len(d)
}

// Less of the urllist implementation should provide the means
// to sort a list of doiitems first by Isodate in ascending
// and in case of identical dates by Title in ascending order.
func (d urllist) Less(i, j int) bool {
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

	return idate.Before(jdate)
}

func (d urllist) Swap(i, j int) {
	d[i], d[j] = d[j], d[i]
}

// mksitemap reads the provided XML files or URLs and generates a
// google sitemap 'urls.txt' file containing the corresponding DOI URLs.
// If an outpath is provided, the file will be created there; default is
// the current working directory.
func mksitemap(xmlFiles []string, outpath string) {
	log.Printf("Parsing %d files\n", len(xmlFiles))

	var urls []doiitem
	for idx, filearg := range xmlFiles {
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

		if len(metadata.Titles) < 1 {
			log.Printf("Could not parse DOI title, skipping '%s'\n", filearg)
			continue
		}
		if len(metadata.Dates) < 1 {
			log.Printf("Could not parse DOI date issued, skipping '%s'\n", filearg)
			continue
		}

		// required to sort list
		curr := doiitem{
			Title:     metadata.Titles[0],
			Shorthash: metadata.Identifier.ID,
			Isodate:   metadata.Dates[0].Value,
		}
		urls = append(urls, curr)
	}

	// sort by date and title ascending
	sort.Sort(urllist(urls))

	var siteurls string
	for _, item := range urls {
		siteurls += fmt.Sprintf("https://doi.gin.g-node.org/%s/\n", item.Shorthash)
	}

	if siteurls == "" {
		log.Printf("Sitemap filecontent empty, skipping empty file creation.")
		return
	}

	fname := "urls.txt"
	if outpath != "" {
		fname = filepath.Join(outpath, fname)
	}

	err := ioutil.WriteFile(fname, []byte(siteurls), 0664)
	if err != nil {
		log.Printf("Error writing sitemap file: %s\n", err.Error())
	}
}

// clisitemap handles command line arguments and passes them
// to the mksitemap function.
// An optional output file path can be passed via the command
// line arguments; default output path is the current working directory.
func clisitemap(cmd *cobra.Command, args []string) {
	var outpath string
	oval, err := cmd.Flags().GetString("out")
	if err != nil {
		log.Printf("-- Error parsing output directory flag: %s\n", err.Error())
	} else if oval != "" {
		outpath = oval
		log.Printf("-- Using output directory '%s'\n", outpath)
	}

	mksitemap(args, outpath)
}
