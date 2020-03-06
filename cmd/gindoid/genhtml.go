package main

import (
	"encoding/xml"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"

	gdtmpl "github.com/G-Node/gin-doi/templates"
	"github.com/G-Node/libgin/libgin"
	humanize "github.com/dustin/go-humanize"
	"github.com/spf13/cobra"
)

const (
	defginurl   = "https://gin.g-node.org"
	defdoibase  = "10.12751/g-node."
	defstoreurl = "https://doid.gin.g-node.org"
)

func isURL(str string) bool {
	if purl, err := url.Parse(str); err == nil {
		if purl.Scheme == "" {
			return false
		}
		return true
	}
	return false
}

func readFileAtPath(path string) ([]byte, error) {
	fp, err := os.Open(path)
	if err != nil {
		return nil, err
	}

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

		// if no DOI found in file, just fall back to the argument number
		fname := fmt.Sprintf("%s.html", strings.ReplaceAll(metadata.Identifier.ID, "/", "_"))
		if metadata.Identifier.ID == "" {
			fmt.Println("WARNING: Couldn't determine DOI. Using generic filename.")
			fname = fmt.Sprintf("%03d-index.html", idx)
		}
		if err := createLandingPage(metadata, fname); err != nil {
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
		return nil, fmt.Errorf("Failed to read metadata for repository %q\n", repopath)
	}
	doiInfo, err := readRepoYAML(infoyml)
	if err != nil {
		return nil, fmt.Errorf("Failed to parse metadata for repository %q\n", repopath)
	}
	return doiInfo, nil
}

func writeHTML(metadata *libgin.RepositoryMetadata) (string, error) {
	funcs := template.FuncMap{
		"Upper":       strings.ToUpper,
		"FunderName":  FunderName,
		"AwardNumber": AwardNumber,
		"AuthorBlock": AuthorBlock,
		"JoinComma":   JoinComma,
	}
	tmpl, err := template.New("doiInfo").Funcs(funcs).Parse(gdtmpl.DOIInfo)
	if err != nil {
		log.Print("Could not parse the DOI Info template")
		return "", err
	}
	tmpl, err = tmpl.New("landingpage").Parse(gdtmpl.LandingPage)
	if err != nil {
		log.Print("Could not parse the DOI template")
		return "", err
	}

	target := metadata.Identifier.ID
	os.MkdirAll(target, 0777)
	filepath := filepath.Join(target, "index.html")
	fp, err := os.Create(filepath)
	if err != nil {
		log.Print("Could not create the DOI index.html")
		return "", err
	}
	defer fp.Close()
	if err := tmpl.Execute(fp, metadata); err != nil {
		log.Print("Could not execute the DOI template")
		return "", err
	}

	fmt.Printf("HTML page saved  %q\n", filepath)
	return filepath, nil
}

// getArchiveSize checks if the DOI is already registered and if it is,
// retrieves the size of the dataset archive.
// If it fails in any way, it returns an empty string.
func getArchiveSize(storeurl string, doibase string, uuid string) string {
	// try both new (doi-based) and old (uuid-based) zip filenames since we
	// currently have both on the server

	doi := doibase + uuid[:6]
	zipfnames := []string{
		strings.ReplaceAll(doi, "/", "_") + ".zip",
		uuid + ".zip",
	}

	var size int64
	for _, zipfname := range zipfnames {
		zipurl, _ := url.Parse(storeurl)
		zipurl.Path = path.Join(doi, zipfname)

		resp, err := http.Get(zipurl.String())
		if err != nil {
			fmt.Printf("Request for archive %q failed: %s\n", zipurl.String(), err.Error())
			continue
		} else if resp.StatusCode != http.StatusOK {
			fmt.Printf("Request for archive %q failed: %s\n", zipurl.String(), resp.Status)
			continue
		}
		size = resp.ContentLength
		return humanize.IBytes(uint64(size))
	}
	return ""
}
