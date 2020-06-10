package main

import (
	"encoding/xml"
	"fmt"
	"html/template"
	"log"
	"os"

	gdtmpl "github.com/G-Node/gin-doi/templates"
	"github.com/G-Node/libgin/libgin"
	"github.com/spf13/cobra"
)

func mkkeywords(cmd *cobra.Command, args []string) {
	keywordMap := make(map[string][]string) // map keywords to DOIs
	fmt.Println("Reading files")
	for idx, filearg := range args {
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

		for _, kw := range metadata.Subjects {
			doilist := keywordMap[kw]
			doilist = append(doilist, metadata.Identifier.ID)
			keywordMap[kw] = doilist
		}
		fmt.Printf(" %d/%d\r", idx+1, len(args))
	}

	fmt.Printf("\nFound %d keywords\n", len(keywordMap))
	fmt.Println("Creating pages")

	for kw, doilist := range keywordMap {
		tmpl, err := template.New(kw).Parse(gdtmpl.Keyword)
		if err != nil {
			log.Printf("Could not parse the keyword page template: %s", err.Error())
			continue
		}

		fp, err := os.Create(fmt.Sprintf("%s.html", kw))
		if err != nil {
			log.Printf("Could not create the keyword page file: %s", err.Error())
			continue
		}
		defer fp.Close()
		data := make(map[string]interface{})
		data["Keyword"] = kw
		data["DOIs"] = doilist
		if err := tmpl.Execute(fp, data); err != nil {
			log.Printf("Error rendering the keyword: %s", err.Error())
			continue
		}
		continue
	}
}
