package main

import (
	"encoding/xml"
	"fmt"
	"io/ioutil"

	"github.com/G-Node/libgin/libgin"
	"github.com/spf13/cobra"
)

// mksitemap reads the provided XML files or URLs and generates a
// google sitemap 'urls.txt' files with the corresponding links.
func mksitemap(cmd *cobra.Command, args []string) {
	fmt.Printf("Parsing %d files\n", len(args))

	var siteurls string
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

		siteurls += fmt.Sprintf("https://doi.gin.g-node.org/%s/\n", metadata.Identifier.ID)
	}

	fname := "urls.txt"
	err := ioutil.WriteFile(fname, []byte(siteurls), 0664)
	if err != nil {
		fmt.Printf("Error writing sitemap file: %s", err.Error())
	}
}
