package main

import (
	"encoding/xml"
	"fmt"
	"net/url"
	"os"
	"path"
	"strings"

	"github.com/G-Node/libgin/libgin"
	"github.com/spf13/cobra"
)

const (
	defginurl   = "https://gin.g-node.org"
	defdoibase  = "10.12751/g-node."
	defstoreurl = "https://doid.gin.g-node.org"
)

func readFileAtPath(path string) ([]byte, error) {
	fp, err := os.Open(path)
	if err != nil {
		return nil, err
	}

	defer fp.Close()

	stat, err := fp.Stat()
	if err != nil {
		return nil, err
	}
	contents := make([]byte, stat.Size())
	_, err = fp.Read(contents)
	return contents, err
}

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

		// if no DOI found in file, just fall back to the argument number
		os.MkdirAll(metadata.Identifier.ID, 0777)
		fname := fmt.Sprintf("%s/index.html", metadata.Identifier.ID)
		if metadata.Identifier.ID == "" {
			fmt.Println("WARNING: Couldn't determine DOI. Using generic filename.")
			fname = fmt.Sprintf("%03d-index.html", idx)
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

func fetchAndParse(ginurl string, repopath string) (*libgin.RepositoryYAML, error) {
	repourl, _ := url.Parse(ginurl)
	repoDatacitePath := path.Join(repopath, "raw", "master", "datacite.yml")
	repourl.Path = repoDatacitePath
	fmt.Printf("Fetching metadata from %s\n", repourl.String())
	infoyml, err := readFileAtURL(repourl.String())
	if err != nil {
		return nil, fmt.Errorf("failed to read metadata for repository %q", repopath)
	}
	doiInfo, err := readRepoYAML(infoyml)
	if err != nil {
		return nil, fmt.Errorf("failed to parse metadata for repository %q", repopath)
	}
	return doiInfo, nil
}
