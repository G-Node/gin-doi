package main

import (
	"fmt"
	"html/template"
	"log"
	"net/url"
	"os"
	"path"
	"path/filepath"

	"github.com/G-Node/libgin/libgin"
	"github.com/spf13/cobra"
)

const defginurl = "https://gin.g-node.org"

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

	fmt.Printf("Generating %d pages\n", len(args))
	var success int
	for idx, repopath := range args {
		fmt.Printf("%3d: %s\n", idx, repopath)
		doiInfo, err := fetchAndParse(ginurl, repopath)
		if err != nil {
			fmt.Println(err.Error())
			continue
		}
		_, err = writeHTML(doiInfo)
		if err != nil {
			fmt.Println(err.Error())
			continue
		}
		// all good
		success++
	}

	fmt.Printf("%d/%d jobs completed successfully\n", success, len(args))
}

func fetchAndParse(ginurl *url.URL, repopath string) (*libgin.DOIRegInfo, error) {
	repourl, _ := url.Parse(ginurl.String()) // make a copy of the base GIN URL
	repoDatacitePath := path.Join(repopath, "raw", "master", "datacite.yml")
	repourl.Path = repoDatacitePath
	fmt.Printf("Fetching metadata from %s\n", repourl.String())
	infoyml, err := readFileAtURL(repourl.String())
	if err != nil {
		return nil, fmt.Errorf("Failed to read metadata for repository %q\n", repopath)
	}
	doiInfo, err := parseDOIInfo(infoyml)
	if err != nil {
		return nil, fmt.Errorf("Failed to parse metadata for repository %q\n", repopath)
	}
	return doiInfo, nil
}

func writeHTML(info *libgin.DOIRegInfo) (string, error) {
	tmpl, err := template.New("landingpage").Parse(landingPageTmpl)
	if err != nil {
		log.Print("Could not parse the DOI template")
		return "", err
	}

	target := info.DOI
	os.MkdirAll(target, 0777)
	filepath := filepath.Join(target, "index.html")
	fp, err := os.Create(filepath)
	if err != nil {
		log.Print("Could not create the DOI index.html")
		return "", err
	}
	defer fp.Close()
	if err := tmpl.Execute(fp, info); err != nil {
		log.Print("Could not execute the DOI template")
		return "", err
	}

	fmt.Printf("HTML page saved  %q\n", filepath)
	return filepath, nil
}
