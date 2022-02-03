package main

import (
	"encoding/xml"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/G-Node/libgin/libgin"
	"github.com/spf13/cobra"
)

// mkhtml reads the provided XML files or URLs and generates the HTML landing
// page for each.
func mkhtml(xmlFiles []string, outpath string) {
	fmt.Printf("Generating %d pages\n", len(xmlFiles))
	var success int
	for idx, filearg := range xmlFiles {
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

		// check missing titles and rightslist; initialize empty
		// and notify without failing to avoid broken html files.
		if len(metadata.Titles) < 1 {
			metadata.Titles = []string{""}
			fmt.Printf("Warning: no titles found in file %q\n", filearg)
		}
		if len(metadata.RightsList) < 1 {
			metadata.RightsList = []libgin.Rights{{Name: "", URL: ""}}
			fmt.Printf("Warning: no Rights found in file %q\n", filearg)
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

		dname := filepath.Join(outpath, metadata.Identifier.ID)
		fname := filepath.Join(outpath, metadata.Identifier.ID, "index.html")
		// If no DOI was found in the file do not create directory and
		// fall back to the argument number.
		if metadata.Identifier.ID == "" {
			fmt.Println("WARNING: Couldn't determine DOI. Using generic filename.")
			fname = filepath.Join(outpath, fmt.Sprintf("%03d-index.html", idx))
		} else if err = os.MkdirAll(dname, 0777); err != nil {
			fmt.Printf("WARNING: Could not create directory: %q", err.Error())
			fname = filepath.Join(outpath, fmt.Sprintf("%s-index.html", metadata.Identifier.ID))
		}

		if err := createLandingPage(metadata, fname, ""); err != nil {
			fmt.Printf("Failed to render landing page for %q: %s\n", filearg, err.Error())
			continue
		}

		fmt.Printf("\t-> %s\n", fname)
		// all good
		success++
	}

	fmt.Printf("%d/%d jobs completed successfully\n", success, len(xmlFiles))
}

// clihtml handles command line arguments and passes them
// to the mkhtml function.
// An optional output file path can be passed via the command
// line arguments; default output path is the current working directory.
func clihtml(cmd *cobra.Command, args []string) {
	var outpath string
	oval, err := cmd.Flags().GetString("out")
	if err != nil {
		log.Printf("-- Error parsing output directory flag: %s\n", err.Error())
	} else if oval != "" {
		outpath = oval
		log.Printf("-- Using output directory '%s'\n", outpath)
	}

	mkhtml(args, outpath)
}
