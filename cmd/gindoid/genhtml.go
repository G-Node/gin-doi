package main

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/G-Node/libgin/libgin"
	humanize "github.com/dustin/go-humanize"
	"github.com/spf13/cobra"
)

const (
	defginurl   = "https://gin.g-node.org"
	defdoibase  = "10.12751/g-node."
	defstoreurl = "https://doid.gin.g-node.org"
)

// mkhtml fetches the metadata file from the configured server for each listed
// repository and generates the html landing page.
// The only configuration line it needs is the address of the GIN web server.
func mkhtml(cmd *cobra.Command, args []string) {
	ginurlstr := libgin.ReadConf("ginurl")
	var ginurl *url.URL
	var err error
	if ginurlstr == "" {
		fmt.Printf("Using default URL for GIN server: %s\n", defginurl)
		ginurlstr = defginurl
		ginurl, _ = url.Parse(ginurlstr)
	} else {
		ginurl, err = url.Parse(ginurlstr)
		if err != nil {
			log.Fatalf("Failed to parse URL for GIN server: %s", ginurlstr)
		}
		fmt.Printf("Using server at %s\n", ginurl)
	}

	doibase := libgin.ReadConf("doibase")
	if doibase == "" {
		fmt.Printf("Using default DOI prefix: %s\n", defdoibase)
		doibase = defdoibase
	}

	var storeurl *url.URL
	storeurlstr := libgin.ReadConf("storeurl")
	if storeurlstr == "" {
		fmt.Printf("Using default store URL: %s\n", defstoreurl)
		storeurl, _ = url.Parse(defstoreurl)
	} else {
		storeurl, err = url.Parse(storeurlstr)
		if err != nil {
			log.Fatalf("Failed to parse registered dataset store URL: %s", storeurlstr)
		}
		fmt.Printf("Using store URL %s\n", storeurl)
	}

	fmt.Printf("Generating %d pages\n", len(args))
	var success int
	for idx, repopath := range args {
		fmt.Printf("%3d: %s\n", idx, repopath)
		doiInfo, err := fetchAndParse(ginurl, repopath)
		if err != nil {
			fmt.Println(err.Error())
			continue
		}

		uuid := makeUUID(repopath)
		metadata := &libgin.RepositoryMetadata{
			YAMLData:         doiInfo,
			SourceRepository: repopath,
			UUID:             uuid,
		}
		metadata.Size = getArchiveSize(storeurl, doibase, uuid)

		_, err = writeHTML(metadata)
		if err != nil {
			fmt.Println(err.Error())
			continue
		}
		// all good
		success++
	}

	fmt.Printf("%d/%d jobs completed successfully\n", success, len(args))
}

func fetchAndParse(ginurl *url.URL, repopath string) (*libgin.RepositoryYAML, error) {
	repourl, _ := url.Parse(ginurl.String()) // make a copy of the base GIN URL
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
	tmpl, err := template.New("doiInfo").Funcs(funcs).Parse(doiInfoTmpl)
	if err != nil {
		log.Print("Could not parse the DOI Info template")
		return "", err
	}
	tmpl, err = tmpl.New("landingpage").Parse(landingPageTmpl)
	if err != nil {
		log.Print("Could not parse the DOI template")
		return "", err
	}

	target := metadata.DOI
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
func getArchiveSize(storeurl *url.URL, doibase string, uuid string) string {
	// try both new (doi-based) and old (uuid-based) zip filenames since we
	// currently have both on the server

	doi := doibase + uuid[:6]
	zipfnames := []string{
		strings.ReplaceAll(doi, "/", "_") + ".zip",
		uuid + ".zip",
	}

	var size int64
	for _, zipfname := range zipfnames {
		zipurl, _ := url.Parse(storeurl.String())
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
