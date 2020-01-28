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
		storeurlstr = defstoreurl
		storeurl, _ = url.Parse(storeurlstr)
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
		// template expects DOIReq that wraps DOIRegInfo and DOIRequestData
		uuid := makeUUID(repopath)
		doiInfo.DOI = doibase + uuid[:6]
		doiInfo.FileSize = getArchiveSize(storeurl, doiInfo.DOI)
		req := &DOIReq{
			DOIInfo:        doiInfo,
			DOIRequestData: &libgin.DOIRequestData{Repository: repopath},
		}
		_, err = writeHTML(req)
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

func writeHTML(req *DOIReq) (string, error) {
	funcs := template.FuncMap{
		"AuthorBlock": AuthorBlock,
	}
	tmpl, err := template.New("landingpage").Funcs(funcs).Parse(landingPageTmpl)
	if err != nil {
		log.Print("Could not parse the DOI template")
		return "", err
	}

	info := req.DOIInfo

	target := info.DOI
	os.MkdirAll(target, 0777)
	filepath := filepath.Join(target, "index.html")
	fp, err := os.Create(filepath)
	if err != nil {
		log.Print("Could not create the DOI index.html")
		return "", err
	}
	defer fp.Close()
	if err := tmpl.Execute(fp, req); err != nil {
		log.Print("Could not execute the DOI template")
		return "", err
	}

	fmt.Printf("HTML page saved  %q\n", filepath)
	return filepath, nil
}

// getArchiveSize checks if the DOI is already registered and if it is,
// retrieves the size of the dataset archive.
// If it fails in any way, it returns an empty string.
func getArchiveSize(storeurl *url.URL, doi string) string {
	zipfname := strings.ReplaceAll(doi, "/", "_") + ".zip"
	zipurl, _ := url.Parse(storeurl.String())
	zipurl.Path = path.Join(doi, zipfname)

	resp, err := http.Get(zipurl.String())
	if err != nil {
		fmt.Printf("Request for archive %q failed: %s\n", zipurl.String(), err.Error())
		return ""
	} else if resp.StatusCode != http.StatusOK {
		fmt.Printf("Request for archive %q failed: %s\n", zipurl.String(), resp.Status)
		return ""
	}
	size := resp.ContentLength
	return humanize.IBytes(uint64(size))
}
