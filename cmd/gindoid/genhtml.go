package main

import (
	"encoding/xml"
	"fmt"
	"os"
	"strings"

	"github.com/G-Node/libgin/libgin"
	"github.com/spf13/cobra"
)

// mkhtml reads the provided XML files or URLs and generates the HTML landing
// page for each.
func mkhtml(cmd *cobra.Command, args []string) {
	fmt.Printf("Generating %d pages\n", len(args))
	var success int
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

		// find URLs in RelatedIdentifiers
		for _, relid := range metadata.RelatedIdentifiers {
			switch u := strings.ToLower(relid.Identifier); {
			case strings.HasPrefix(u, "https://gin.g-node.org/doi/"):
				// fork URL
				metadata.ForkRepository = strings.TrimPrefix(relid.Identifier, "https://gin.g-node.org/")
			case strings.HasPrefix(u, "https://web.gin.g-node.org/doi"):
				// fork URL (old)
				metadata.ForkRepository = strings.TrimPrefix(relid.Identifier, "https://web.gin.g-node.org/")
			case strings.HasPrefix(u, "https://gin.g-node.org/"):
				// repo URL
				metadata.SourceRepository = strings.TrimPrefix(relid.Identifier, "https://gin.g-node.org/")
			case strings.HasPrefix(u, "https://web.gin.g-node.org/"):
				// repo URL (old)
				metadata.SourceRepository = strings.TrimPrefix(relid.Identifier, "https://web.gin.g-node.org/")
			}
		}

		fname := fmt.Sprintf("%s/index.html", metadata.Identifier.ID)
		// If no DOI was found in the file do not create directory and
		// fall back to the argument number.
		if metadata.Identifier.ID == "" {
			fmt.Println("WARNING: Couldn't determine DOI. Using generic filename.")
			fname = fmt.Sprintf("%03d-index.html", idx)
		} else if err = os.MkdirAll(metadata.Identifier.ID, 0777); err != nil {
			fmt.Printf("WARNING: Could not create directory: %q", err.Error())
			fname = fmt.Sprintf("%s-index.html", metadata.Identifier.ID)
		}
		if err := createLandingPage(metadata, fname, ""); err != nil {
			fmt.Printf("Failed to render landing page for %q: %s\n", filearg, err.Error())
			continue
		}

		fmt.Printf("\t-> %s\n", fname)
		// all good
		success++
	}

	fmt.Printf("%d/%d jobs completed successfully\n", success, len(args))
}
