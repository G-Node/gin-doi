package main

import (
	"bytes"
	"html/template"
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
	tmpl.Execute(&b, data)
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
	tmpl.Execute(&b, data)
	if checkurl != b.String() {
		t.Fatalf("Error dynamic URL; got: '%s'", b.String())
	}
}
