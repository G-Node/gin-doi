package main

import (
	"bytes"
	"fmt"
	"html/template"
	"net/url"
	"testing"

	"github.com/G-Node/libgin/libgin"
)

func TestRequestFailureTemplate(t *testing.T) {
	regRequest := new(RegistrationRequest)
	regRequest.Message = template.HTML(msgInvalidRequest)
	regRequest.Metadata = new(libgin.RepositoryMetadata)
	regRequest.DOIRequestData = new(libgin.DOIRequestData) // Source repo required to render fail page
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

	// create local test file server
	server := serveDataciteServer()
	defer server.Close()

	// check local test server works
	serverURL, err := url.Parse(server.URL)
	if err != nil {
		t.Fatalf("Could not parse server URL: %q", serverURL)
	}

	// read datacite.yml template for test
	infoURL := fmt.Sprintf("%s/reference-dc-yml", server.URL)
	infoyml, err := readFileAtURL(infoURL)
	if err != nil {
		t.Fatalf("Failed to retrieve datacite.yml from GIN")
	}
	doiInfo, err := readRepoYAML(infoyml)
	if err != nil {
		t.Fatalf("Failed to read datacite.yaml")
	}
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

	w := new(bytes.Buffer)
	if err := tmpl.Execute(w, regRequest); err != nil {
		t.Log(w.String()) // Print the rendered output
		t.Fatalf("Failed to execute RequestPage: %s", err.Error())
	}
}

func TestRequestResultTemplate(t *testing.T) {
	tmpl, err := prepareTemplates("RequestResult")
	if err != nil {
		t.Fatalf("Failed to parse RequestResult template: %s", err.Error())
	}

	resData := new(reqResultData)

	// failure
	resData.Success = false
	resData.Level = "error"
	resData.Message = template.HTML(msgSubmitFailed)

	w := new(bytes.Buffer)
	if err := tmpl.Execute(w, resData); err != nil {
		t.Log(w.String()) // Print the rendered output
		t.Fatalf("Failed to execute RequestResult: %s", err.Error())
	}

	// warning
	resData.Success = true
	resData.Level = "warning"
	resData.Message = template.HTML(msgSubmitError)

	w = new(bytes.Buffer)
	if err := tmpl.Execute(w, resData); err != nil {
		t.Log(w.String()) // Print the rendered output
		t.Fatalf("Failed to execute RequestResult: %s", err.Error())
	}

	// success
	message := fmt.Sprintf(msgServerIsArchiving, "test/DOI.xyz")
	resData.Success = true
	resData.Level = "success"
	resData.Message = template.HTML(message)

	w = new(bytes.Buffer)
	if err := tmpl.Execute(w, resData); err != nil {
		t.Log(w.String()) // Print the rendered output
		t.Fatalf("Failed to execute RequestResult: %s", err.Error())
	}
}

func TestLandingPageTemplate(t *testing.T) {
	tmpl, err := prepareTemplates("DOIInfo", "LandingPage")
	if err != nil {
		t.Fatalf("Failed to parse DOIInfo, LandingPage templates: %s", err.Error())
	}
	// create local test file server
	server := serveDataciteServer()
	defer server.Close()

	// check local test server works
	serverURL, err := url.Parse(server.URL)
	if err != nil {
		t.Fatalf("Could not parse server URL: %q", serverURL)
	}

	// read datacite.yml template for test
	infoURL := fmt.Sprintf("%s/reference-dc-yml", server.URL)
	infoyml, err := readFileAtURL(infoURL)
	if err != nil {
		t.Fatalf("Failed to retrieve datacite.yml from GIN")
	}
	doiInfo, err := readRepoYAML(infoyml)
	if err != nil {
		t.Fatalf("Failed to read datacite.yml")
	}
	metadata := new(libgin.RepositoryMetadata)
	metadata.YAMLData = doiInfo
	metadata.DataCite = libgin.NewDataCiteFromYAML(doiInfo)
	metadata.SourceRepository = "test/repository"
	metadata.ForkRepository = "doi/repository"

	w := new(bytes.Buffer)
	if err := tmpl.Execute(w, metadata); err != nil {
		t.Log(w.String()) // Print the rendered output
		t.Fatalf("Failed to execute LandingPage: %s", err.Error())
	}
}

func TestKeywordIndexTemplate(t *testing.T) {
	tmpl, err := prepareTemplates("KeywordIndex")
	if err != nil {
		t.Fatalf("Failed to parse KeywordIndex templates: %s", err.Error())
	}
	// create local test file server
	server := serveDataciteServer()
	defer server.Close()

	// check local test server works
	serverURL, err := url.Parse(server.URL)
	if err != nil {
		t.Fatalf("Could not parse server URL: %q", serverURL)
	}

	// read datacite.yml template for test
	infoURL := fmt.Sprintf("%s/reference-dc-yml", server.URL)
	infoyml, err := readFileAtURL(infoURL)
	if err != nil {
		t.Fatalf("Failed to retrieve datacite.yml from GIN")
	}
	doiInfo, err := readRepoYAML(infoyml)
	if err != nil {
		t.Fatalf("Failed to read datacite.yml")
	}
	metadata := new(libgin.RepositoryMetadata)
	metadata.YAMLData = doiInfo
	metadata.DataCite = libgin.NewDataCiteFromYAML(doiInfo)
	metadata.SourceRepository = "test/repository"
	metadata.ForkRepository = "doi/repository"

	data := make(map[string]interface{})
	data["KeywordList"] = []string{"a", "b", "anotherkeyword"}
	keywordMap := make(map[string][]*libgin.RepositoryMetadata, 3)
	keywordMap["a"] = []*libgin.RepositoryMetadata{metadata}
	keywordMap["b"] = []*libgin.RepositoryMetadata{metadata}
	keywordMap["anotherkeyword"] = []*libgin.RepositoryMetadata{metadata}
	data["KeywordMap"] = keywordMap

	w := new(bytes.Buffer)
	if err := tmpl.Execute(w, data); err != nil {
		t.Log(w.String()) // Print the rendered output
		t.Fatalf("Failed to execute KeywordIndex: %s", err.Error())
	}
}

func TestKeywordTemplate(t *testing.T) {
	tmpl, err := prepareTemplates("Keyword")
	if err != nil {
		t.Fatalf("Failed to parse Keyword templates: %s", err.Error())
	}

	// create local test file server
	server := serveDataciteServer()
	defer server.Close()

	// check local test server works
	serverURL, err := url.Parse(server.URL)
	if err != nil {
		t.Fatalf("Could not parse server URL: %q", serverURL)
	}

	// read datacite.yml template for test
	infoURL := fmt.Sprintf("%s/reference-dc-yml", server.URL)
	infoyml, err := readFileAtURL(infoURL)
	if err != nil {
		t.Fatalf("Failed to retrieve datacite.yml from GIN")
	}
	doiInfo, err := readRepoYAML(infoyml)
	if err != nil {
		t.Fatalf("Failed to read datatcite.yml")
	}
	metadata := new(libgin.RepositoryMetadata)
	metadata.YAMLData = doiInfo
	metadata.DataCite = libgin.NewDataCiteFromYAML(doiInfo)
	metadata.SourceRepository = "test/repository"
	metadata.ForkRepository = "doi/repository"

	data := make(map[string]interface{})
	data["KeywordList"] = []string{"a", "b", "anotherkeyword"}
	data["Keyword"] = "test"
	data["Datasets"] = []*libgin.RepositoryMetadata{metadata}

	w := new(bytes.Buffer)
	if err := tmpl.Execute(w, data); err != nil {
		t.Log(w.String()) // Print the rendered output
		t.Fatalf("Failed to execute Keyword: %s", err.Error())
	}

}
