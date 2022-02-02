package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
)

const invalidTestDataciteXML = `sometext`
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
    <subject>Test keyword</subject>
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

// serveDataciteXMLserver provides a local test server for Datacite xml handling
func serveDataciteXMLserver() *httptest.Server {
	mux := http.NewServeMux()
	mux.HandleFunc("/non-xml", func(rw http.ResponseWriter, req *http.Request) {
		rw.WriteHeader(http.StatusOK)
		_, err := rw.Write([]byte(invalidTestDataciteXML))
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

	return httptest.NewServer(mux)
}
