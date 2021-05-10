package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/G-Node/libgin/libgin"
	"github.com/spf13/cobra"
)

// cleanupGINURL returns owner and repository of a GIN
// repository datacite URL string. Example:
// https://gin.g-node.org/G-Node/doi_deployment_test/raw/master/datacite.yml
func cleanupGINURL(input string) (string, error) {
	cleanstr := strings.Replace(input, "https://gin.g-node.org/", "", -1)
	cleanstr = strings.Replace(cleanstr, "/raw/master/datacite.yml", "", -1)
	out := strings.Split(cleanstr, "/")

	if len(out) != 2 {
		return "", fmt.Errorf("could not parse URL: %s", input)
	}
	return out[1], nil
}

// mkxml reads one or more datacite YAML files from GIN, a provided URL or from 
// a direct file and generates a Datacite XML file for each.
// Reading files from GIN requires only the repository owner and the repository name
// of the GIN repository prefixed with GIN in the format "GIN:[owner]/[repository]"
func mkxml(cmd *cobra.Command, args []string) {
	fmt.Printf("Generating %d xml files\n", len(args))
	var success int
	for idx, filearg := range args {
		fmt.Printf("%3d: %s\n", idx, filearg)
		var contents []byte
		var err error
		var repoName string
		if isURL(filearg) {
			repoName, err = cleanupGINURL(filearg)
			if err != nil {
				fmt.Printf("failed to cleanup GIN datacite URL: %s", err.Error())
			}
			contents, err = readFileAtURL(filearg)
		} else {
			contents, err = readFileAtPath(filearg)
		}
		if err != nil {
			fmt.Printf("Failed to read file at %q: %s\n", filearg, err.Error())
			continue
		}

		dataciteContent, err := readRepoYAML(contents)
		if err != nil {
			fmt.Print("DOI file invalid")
			continue
		}

		datacite := libgin.NewDataCiteFromYAML(dataciteContent)

		// Create storage directory
		if repoName == "" {
			repoName = fmt.Sprintf("index-%03d", idx)
		}
		fname := filepath.Join(repoName, "doi.xml")
		if err = os.MkdirAll(repoName, 0777); err != nil {
			fmt.Printf("WARNING: Could not create directory %s: %q", repoName, err.Error())
			fname = fmt.Sprintf("%s-doi.xml", repoName)
		}

		fp, err := os.Create(fname)
		if err != nil {
			// XML Creation failed; return with error
			fmt.Printf("Failed to create the XML metadata file: %s", err)
			continue
		}
		defer fp.Close()

		data, err := datacite.Marshal()
		if err != nil {
			fmt.Printf("Failed to render the XML metadata file: %s", err)
			continue
		}
		_, err = fp.Write([]byte(data))
		if err != nil {
			fmt.Printf("Failed to write the metadata XML file: %s", err)
			continue
		}

		fmt.Printf("\t-> %s\n", fname)
		// all good
		success++
	}

	fmt.Printf("%d/%d jobs completed successfully\n", success, len(args))
}
