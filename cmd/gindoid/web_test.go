package main

import (
	"bytes"
	"html/template"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestInjectDynamicGINURL checks that the function injectDynamicGINURL
// properly parses the gin server URL into an HTML template.
func TestInjectDynamicGINURL(t *testing.T) {
	stringbase := "url: "
	templatestr := stringbase + "{{GINServerURL}}"
	var b bytes.Buffer
	var data interface{}

	// When left empty, test that the default gin server url is parsed into the template
	defaulturl := "https://gin.g-node.org"
	checkurl := stringbase + defaulturl

	tmpl, err := template.New("Test").Funcs(tmplfuncs).Parse(templatestr)
	if err != nil {
		t.Fatalf("Failed to parse test template: %s", err.Error())
	}
	tmpl = injectDynamicGINURL(tmpl, "")
	err = tmpl.Execute(&b, data)
	if err != nil {
		t.Fatalf("Failed to execute test template: %s", err.Error())
	}
	if checkurl != b.String() {
		t.Fatalf("Error default URL; got: '%s'", b.String())
	}

	// When available, test that the provided url is parsed into the template
	b.Reset()
	dynamicurl := "https://dev.g-node.org"
	checkurl = stringbase + dynamicurl

	tmpl, err = template.New("Test").Funcs(tmplfuncs).Parse(templatestr)
	if err != nil {
		t.Fatalf("Failed to parse test template: %s", err.Error())
	}
	tmpl = injectDynamicGINURL(tmpl, dynamicurl)
	err = tmpl.Execute(&b, data)
	if err != nil {
		t.Fatalf("Failed to execute test template: %s", err.Error())
	}
	if checkurl != b.String() {
		t.Fatalf("Error dynamic URL; got: '%s'", b.String())
	}
}

func TestRenderResult(t *testing.T) {
	cfg := Configuration{}
	resData := reqResultData{}
	w := httptest.NewRecorder()

	renderResult(w, &resData, &cfg)

	content := w.Body.Bytes()
	if !strings.Contains(string(content), "DOI request failed") {
		t.Fatal("Did not retrieve DOI request fail page")
	}
}
