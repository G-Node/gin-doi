package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
)

const empty = ``
const plainText = `plaintext`
const validTestDataciteXML = `<?xml version="1.0" encoding="UTF-8"?>
<resource xmlns="http://datacite.org/schema/kernel-4" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:schemaLocation="http://datacite.org/schema/kernel-4 http://schema.datacite.org/meta/kernel-4.3/metadata.xsd">
  <identifier identifierType="DOI">10.12751/g-node.noex1st</identifier>
  <creators>
    <creator>
      <creatorName>TestLastname, GivenName</creatorName>
      <nameIdentifier schemeURI="http://orcid.org/" nameIdentifierScheme="ORCID">0000-000X-XXXX-XXXX</nameIdentifier>
      <affiliation>Test Affiliation</affiliation>
    </creator>
  </creators>
  <titles>
    <title>Test title</title>
  </titles>
  <descriptions>
    <description descriptionType="Abstract">Test abstract</description>
  </descriptions>
  <rightsList>
    <rights rightsURI="https://opensource.org/licenses/BSD-3-Clause/">BSD-3-Clause</rights>
  </rightsList>
  <subjects>
    <subject>Test_keyword</subject>
  </subjects>
  <relatedIdentifiers>
    <relatedIdentifier relatedIdentifierType="URL" relationType="IsVariantFormOf">https://doi.gin.g-node.org/10.12751/g-node.noex1st/10.12751_g-node.noex1st.zip</relatedIdentifier>
  </relatedIdentifiers>
  <fundingReferences>
    <fundingReference>
      <funderName>Test Funderreference</funderName>
    </fundingReference>
  </fundingReferences>
  <contributors>
    <contributor contributorType="HostingInstitution">
      <contributorName>German Neuroinformatics Node</contributorName>
    </contributor>
  </contributors>
  <publisher>G-Node</publisher>
  <publicationYear>2021</publicationYear>
  <dates>
    <date dateType="Issued">2021-10-27</date>
  </dates>
  <language>eng</language>
  <resourceType resourceTypeGeneral="Dataset">Dataset</resourceType>
  <sizes>
    <size>8.1 MiB</size>
  </sizes>
  <version>1.0</version>
</resource>
`

const emptyTestDataciteXML = `<?xml version="1.0" encoding="UTF-8"?>
<resource xmlns="http://datacite.org/schema/kernel-4" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:schemaLocation="http://datacite.org/schema/kernel-4 http://schema.datacite.org/meta/kernel-4.3/metadata.xsd">
  <version>1.0</version>
</resource>
`

const validTestDataciteYML = `
authors:
  -
    firstname: "aaa"
    lastname: "MadamA"
    affiliation: "Department of Test simple author"

# A title to describe the published resource.
title: "Doi test"

# Do not edit or remove the following line
templateversion: 1.2
`

const referenceDataciteYML = `# Metadata for DOI registration according to DataCite Metadata Schema 4.1.
# For detailed schema description see https://doi.org/10.5438/0014

## Required fields

# The main researchers involved. Include digital identifier (e.g., ORCID)
# if possible, including the prefix to indicate its type.
authors:
  -
    firstname: "GivenName1"
    lastname: "FamilyName1"
    affiliation: "Affiliation1"
    id: "ORCID:0000-0001-2345-6789"
  -
    firstname: "GivenName2"
    lastname: "FamilyName2"
    affiliation: "Affiliation2"
    id: "ResearcherID:X-1234-5678"
  -
    firstname: "GivenName3"
    lastname: "FamilyName3"

# A title to describe the published resource.
title: "Example Title"

# Additional information about the resource, e.g., a brief abstract.
description: |
  Example description
  that can contain linebreaks
  but has to maintain indentation.

# Lit of keywords the resource should be associated with.
# Give as many keywords as possible, to make the resource findable.
keywords:
  - Neuroscience
  - Keyword2
  - Keyword3

# License information for this resource. Please provide the license name and/or a link to the license.
# Please add also a corresponding LICENSE file to the repository.
license:
  name: "Creative Commons CC0 1.0 Public Domain Dedication"
  url: "https://creativecommons.org/publicdomain/zero/1.0/"



## Optional Fields

# Funding information for this resource.
# Separate funder name and grant number by comma.
funding:
  - "DFG, AB1234/5-6"
  - "EU, EU.12345"


# Related publications. reftype might be: IsSupplementTo, IsDescribedBy, IsReferencedBy.
# Please provide digital identifier (e.g., DOI) if possible.
# Add a prefix to the ID, separated by a colon, to indicate the source.
# Supported sources are: DOI, arXiv, PMID
# In the citation field, please provide the full reference, including title, authors, journal etc.
references:
  -
    id: "doi:10.xxx/zzzz"
    reftype: "IsSupplementTo"
    citation: "Citation1"
  -
    id: "arxiv:mmmm.nnnn"
    reftype: "IsSupplementTo"
    citation: "Citation2"
  -
    id: "pmid:nnnnnnnn"
    reftype: "IsReferencedBy"
    citation: "Citation3"


# Resource type. Default is Dataset, other possible values are Software, Image, Text.
resourcetype: Dataset

# Do not edit or remove the following line
templateversion: 1.2
`

// serveDataciteserver provides a local test server for Datacite xml handling
func serveDataciteServer() *httptest.Server {
	mux := http.NewServeMux()
	mux.HandleFunc("/empty", func(rw http.ResponseWriter, req *http.Request) {
		rw.WriteHeader(http.StatusOK)
		_, err := rw.Write([]byte(empty))
		if err != nil {
			fmt.Printf("could not write valid response: %q", err.Error())
		}
	})
	mux.HandleFunc("/non-xml", func(rw http.ResponseWriter, req *http.Request) {
		rw.WriteHeader(http.StatusOK)
		_, err := rw.Write([]byte(plainText))
		if err != nil {
			fmt.Printf("Could not write invalid response: %q", err.Error())
		}
	})
	mux.HandleFunc("/xml", func(rw http.ResponseWriter, req *http.Request) {
		rw.WriteHeader(http.StatusOK)
		_, err := rw.Write([]byte(validTestDataciteXML))
		if err != nil {
			fmt.Printf("could not write valid response: %q", err.Error())
		}
	})
	mux.HandleFunc("/empty-xml", func(rw http.ResponseWriter, req *http.Request) {
		rw.WriteHeader(http.StatusOK)
		_, err := rw.Write([]byte(emptyTestDataciteXML))
		if err != nil {
			fmt.Printf("could not write valid response: %q", err.Error())
		}
	})
	mux.HandleFunc("/dc-yml", func(rw http.ResponseWriter, req *http.Request) {
		rw.WriteHeader(http.StatusOK)
		_, err := rw.Write([]byte(validTestDataciteYML))
		if err != nil {
			fmt.Printf("could not write valid response: %q", err.Error())
		}
	})
	mux.HandleFunc("/reference-dc-yml", func(rw http.ResponseWriter, req *http.Request) {
		rw.WriteHeader(http.StatusOK)
		_, err := rw.Write([]byte(referenceDataciteYML))
		if err != nil {
			fmt.Printf("could not write valid response: %q", err.Error())
		}
	})

	return httptest.NewServer(mux)
}
