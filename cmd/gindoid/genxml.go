package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/G-Node/libgin/libgin"
	"github.com/spf13/cobra"
)

// getGINDataciteURL returns a full URL to the datacite file at
// the root of a GIN repository where owner and repository name
// were provided as a single input string.
func getGINDataciteURL(input string) (string, error) {
	inputslice := strings.Split(input, "/")
	if len(inputslice) != 2 {
		return "", fmt.Errorf("could not parse gin repo string %s", input)
	}

	ginprefix := "https://gin.g-node.org/"
	ginpostfix := "/raw/master/datacite.yml"
	out := fmt.Sprintf("%s%s%s", ginprefix, input, ginpostfix)

	return out, nil
}

// mkxml reads one or more datacite YAML files from GIN, a provided URL or from
// a direct file and generates a Datacite XML file for each.
// Reading files from GIN requires only the repository owner and the repository name
// of the GIN repository prefixed with GIN in the format "GIN:[owner]/[repository]"
func mkxml(ymlFiles []string, outpath string) {
	fmt.Printf("Generating %d xml files\n", len(ymlFiles))
	var success int
	for idx, filearg := range ymlFiles {
		fmt.Printf("%3d: %s\n", idx, filearg)
		var contents []byte
		var err error
		var repoName string

		if strings.HasPrefix(filearg, "GIN:") {
			repostring := strings.Replace(filearg, "GIN:", "", 1)
			ginurl, err := getGINDataciteURL(repostring)
			if err != nil {
				fmt.Printf("Failed to parse GIN specific repo string: %s\n", filearg)
				continue
			}
			repodata := strings.Split(repostring, "/")
			if len(repodata) == 2 {
				repoName = repodata[1]
			}
			filearg = ginurl
		}

		if isURL(filearg) {
			contents, err = readFileAtURL(filearg)
		} else {
			contents, err = readFileAtPath(filearg)
		}
		if err != nil {
			fmt.Printf("Failed to read file at %q: %s\n", filearg, err.Error())
			continue
		}

		// skip empty files
		if string(contents) == "" {
			fmt.Printf("File %q is empty, skipping\n", filearg)
			continue
		}

		dataciteContent, err := readRepoYAML(contents)
		if err != nil {
			fmt.Printf("DOI file invalid: %s\n", err.Error())
			continue
		}

		// Add datacite quality checks and notify but carry on
		if msgs := validateDataCite(dataciteContent); len(msgs) > 0 {
			fmt.Printf("DOI file contains validation issues: %s\n", strings.Join(msgs, "; "))
		}

		// avoid panic on missing license
		if dataciteContent.License == nil {
			dataciteContent.License = &libgin.License{}
			fmt.Print("DOI file does not provide a License\n")
		}

		// Permit DOI references without prefix without updating the libgin library;
		// this code snippet can be removed if the corresponding libgin function
		// (libgin.DataCite.AddReference) is updated and a new library version is released.
		// This handling takes care of references containing only DOI URLs as well
		// as improperly formatted DOI references entries that end with a proper DOI URL
		// e.g: 'id: "doi:  https://doi.org/some-doi"'.
		doiSplit := "https://doi.org/"
		for idx, ref := range dataciteContent.References {
			if strings.Contains(ref.ID, doiSplit) {
				fmt.Printf("Updating DOI reference ID %q\n", ref.ID)
				doiID := strings.Split(ref.ID, doiSplit)
				dataciteContent.References[idx].ID = fmt.Sprintf("doi:%s", doiID[1])
			}
		}

		datacite := libgin.NewDataCiteFromYAML(dataciteContent)

		// Create storage directory
		if repoName == "" {
			repoName = fmt.Sprintf("index-%03d", idx)
		}
		dirname := filepath.Join(outpath, repoName)
		fname := filepath.Join(outpath, repoName, "doi.xml")
		if err = os.MkdirAll(dirname, 0777); err != nil {
			fmt.Printf("WARNING: Could not create directory %s: %q", dirname, err.Error())
			fname = fmt.Sprintf("%s-doi.xml", repoName)
		}

		fp, err := os.Create(fname)
		if err != nil {
			fmt.Printf("Failed to create the metadata XML file: %s", err)
			continue
		}
		defer fp.Close()

		data, err := datacite.Marshal()
		if err != nil {
			fmt.Printf("Failed to render the metadata XML file: %s", err)
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

	fmt.Printf("%d/%d jobs completed successfully\n", success, len(ymlFiles))
	fmt.Print("\nRemember to add the G-Node identifier and check and adjust sizes and publication dates\n")
}

// clixml handles command line arguments and passes them
// to the mkxml function.
// An optional output file path can be passed via the command
// line arguments; default output path is the current working directory.
func clixml(cmd *cobra.Command, args []string) {
	var outpath string
	oval, err := cmd.Flags().GetString("out")
	if err != nil {
		fmt.Printf("-- Error parsing output directory flag: %s\n", err.Error())
	} else if oval != "" {
		outpath = oval
		fmt.Printf("-- Using output directory '%s'\n", outpath)
	}

	mkxml(args, outpath)
}
