package main

import (
	"encoding/xml"
	"fmt"
	"log"
	"os"
	"sort"
	"strings"

	"github.com/G-Node/libgin/libgin"
	"github.com/spf13/cobra"
)

func mkkeywords(cmd *cobra.Command, args []string) {
	// map keywords to DOIs
	keywordMap := make(map[string][]*libgin.RepositoryMetadata)

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

		// exclude old keywords of re-published datasets
		var exclude bool
		for _, relid := range datacite.RelatedIdentifiers {
			if strings.Contains(strings.ToLower(relid.RelationType), "ispreviousversionof") {
				exclude = true
			}
		}
		if exclude {
			fmt.Printf("Excluding previous version dataset (%s)\n", datacite.Identifier.ID)
			continue
		}

		metadata := &libgin.RepositoryMetadata{
			DataCite: datacite,
		}

		for _, kw := range *metadata.Subjects {
			kw = KeywordPath(kw)
			datasets := keywordMap[kw]
			datasets = append(datasets, metadata)
			keywordMap[kw] = datasets
		}
		fmt.Printf(" %d/%d\r", idx+1, len(args))
	}

	fmt.Printf("\nFound %d keywords\n", len(keywordMap))
	fmt.Println("Creating pages")

	for kw, datasets := range keywordMap {
		tmpl, err := prepareTemplates("Keyword")
		if err != nil {
			continue
		}
		// use a "keywords" root directory
		err = os.MkdirAll(fmt.Sprintf("keywords/%s", kw), 0777)
		if err != nil {
			log.Printf("Could not create the keyword page dir: %s", err.Error())
			continue
		}

		fp, err := os.Create(fmt.Sprintf("keywords/%s/index.html", kw))
		if err != nil {
			log.Printf("Could not create the keyword page file: %s", err.Error())
			continue
		}
		defer fp.Close()
		data := make(map[string]interface{})
		data["Keyword"] = kw
		// Sort by date, lex order, which for ISO date strings should work fine
		sort.Slice(datasets, func(i, j int) bool {
			return datasets[i].Dates[0].Value > datasets[j].Dates[0].Value
		})
		data["Datasets"] = datasets
		if err := tmpl.Execute(fp, data); err != nil {
			log.Printf("Error rendering the keyword: %s", err.Error())
			continue
		}
		continue
	}

	// make keyword index page
	keywordList := make([]string, 0, len(keywordMap))

	// collect keywords in slice and sort by the number of datasets for each
	for kw := range keywordMap {
		keywordList = append(keywordList, kw)
	}
	sort.Slice(keywordList, func(i, j int) bool {
		ilen := len(keywordMap[keywordList[i]])
		jlen := len(keywordMap[keywordList[j]])
		if ilen == jlen {
			// sort alphabetically
			return keywordList[i] < keywordList[j]
		}
		return ilen > jlen
	})

	data := make(map[string]interface{})
	data["KeywordList"] = keywordList
	data["KeywordMap"] = keywordMap
	tmpl, err := prepareTemplates("KeywordIndex")
	if err != nil {
		return
	}
	fp, err := os.Create("keywords/index.html")
	if err != nil {
		log.Printf("Could not create the keyword list page file: %s", err.Error())
		return
	}
	defer fp.Close()

	if err := tmpl.Execute(fp, data); err != nil {
		log.Printf("Error rendering keyword list page: %s", err.Error())
	}
}
