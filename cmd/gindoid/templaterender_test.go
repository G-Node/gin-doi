package main

import (
	"bytes"
	"html/template"
	"testing"

	"github.com/G-Node/libgin/libgin"
)

func TestRequestFailureTemplate(t *testing.T) {
	regRequest := new(RegistrationRequest)
	regRequest.Message = template.HTML(msgInvalidRequest)
	regRequest.Metadata = new(libgin.RepositoryMetadata)
	tmpl, err := prepareTemplates("RequestFailurePage")
	if err != nil {
		t.Fatalf("Failed to parse RequestFailureP template: %s", err.Error())
	}

	w := new(bytes.Buffer)
	if err := tmpl.Execute(w, regRequest); err != nil {
		t.Log(w.String()) // Print the rendered output
		t.Fatalf("Failed to execute RequestFailurePage: %s", err.Error())
	}
}

func TestRequestPageTemplate(t *testing.T) {
	tmpl, err := prepareTemplates("DOIInfo", "RequestPage")
	if err != nil {
		t.Fatalf("Failed to parse DOIInfo, RequestPage templates: %s", err.Error())
	}
	// read datacite.yml template for test
	infoyml, err := readFileAtURL("https://gin.g-node.org/G-Node/Info/raw/master/datacite.yml")
	if err != nil {
		t.Fatalf("Failed to retrieve datacite.yml from GIN")
	}
	doiInfo, err := readRepoYAML(infoyml)
	regRequest := new(RegistrationRequest)
	regRequest.DOIRequestData = &libgin.DOIRequestData{
		Username:   "testuser",
		Realname:   "Test User",
		Repository: "user/test",
		Email:      "doitest@example.org",
	}
	regRequest.Metadata = new(libgin.RepositoryMetadata)
	regRequest.Metadata.YAMLData = doiInfo
	regRequest.Metadata.DataCite = libgin.NewDataCiteFromYAML(doiInfo)
	regRequest.Metadata.SourceRepository = regRequest.DOIRequestData.Repository
	regRequest.Metadata.ForkRepository = "" // not forked yet

	err = tmpl.Execute(w, regRequest)
	if err := tmpl.Execute(w, regRequest); err != nil {
		t.Log(w.String()) // Print the rendered output
		t.Fatalf("Failed to execute RequestPage: %s", err.Error())
	}
}
