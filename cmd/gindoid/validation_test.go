package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/G-Node/libgin/libgin"
)

func writeTmpFile(filename string, content string) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = io.WriteString(file, content)
	if err != nil {
		return err
	}
	return file.Sync()
}

func TestLicenseFromFile(t *testing.T) {
	tmpDir, err := ioutil.TempDir("", "test_gindoi_licfromfile")
	if err != nil {
		t.Fatalf("Error creating tmp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	licfile := filepath.Join(tmpDir, "lic.json")
	licURL := "lic url"
	content := fmt.Sprintf(`[{"URL": "%s", "Name":  "lic name", "Alias": ["alias name"]}]`, licURL)
	err = writeTmpFile(licfile, content)
	if err != nil {
		t.Fatalf("Error creating json file: '%s'", err.Error())
	}

	liclist, err := licenseFromFile(licfile)
	if err != nil {
		t.Fatalf("Could not load custom license file: '%s'", err.Error())
	}
	if len(liclist) != 1 {
		t.Fatalf("Unexpected license list length: '%d'", len(liclist))
	}
	if licURL != liclist[0].URL {
		t.Fatalf("Unexpected license content: '%s'/'%s'", licURL, liclist[0].URL)
	}
}

func TestReadCommonLicenses(t *testing.T) {
	// check loading default licenses
	liclist := ReadCommonLicenses()
	// there should always be more than 2 default licenses
	if len(liclist) < 2 {
		t.Fatalf("Could not read default licenses")
	}

	// provide custom license file and check common licenses are loaded from there
	tmpDir, err := ioutil.TempDir("", "test_gindoi_readCommonLicense")
	if err != nil {
		t.Fatalf("Error creating tmp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	licfile := filepath.Join(tmpDir, "doi-licenses.json")
	licURL := "lic url"
	content := fmt.Sprintf(`[{"URL": "%s", "Name":  "lic name", "Alias": ["alias name"]}]`, licURL)
	err = writeTmpFile(licfile, content)
	if err != nil {
		t.Fatalf("Error creating json file: '%s'", err.Error())
	}

	err = os.Setenv("configdir", tmpDir)
	if err != nil {
		t.Fatalf("Error setting environment: %s", err.Error())
	}
	liclist = ReadCommonLicenses()
	if len(liclist) != 1 {
		t.Fatalf("Error reading custom license file")
	}
	if licURL != liclist[0].URL {
		t.Fatalf("Unexpected license content: '%s'/'%s'", licURL, liclist[0].URL)
	}
}

func TestLicFromURL(t *testing.T) {
	liclist := ReadCommonLicenses()

	// test license URL not in common license list
	licURL := "I AM NOT HERE"
	_, ok := licFromURL(liclist, licURL)
	if ok {
		t.Fatalf("License URL '%s' should not have been found", licURL)
	}

	// test finding deviating character case license URL
	licName := "Creative Commons Attribution 4.0 International Public License"
	licURL = " https://creativecommons.org/licenses/BY/4.0 "
	lic, ok := licFromURL(liclist, licURL)
	if !ok {
		t.Fatalf("Error finding case insensitive URL: '%s'", licURL)
	}
	if lic.Name != licName {
		t.Fatalf("Found invalid license: '%s' expected '%s'", lic.Name, licName)
	}

	// test finding long suffix version of license URL
	licURL = "https://creativecommons.org/licenses/by/4.0/legalcode"
	lic, ok = licFromURL(liclist, licURL)
	if !ok {
		t.Fatalf("Error finding suffix version URL: '%s'", licURL)
	}
	if lic.Name != licName {
		t.Fatalf("Found invalid license: '%s' expected '%s'", lic.Name, licName)
	}
}

func TestLicFromName(t *testing.T) {
	liclist := ReadCommonLicenses()

	// test license not found
	licName := "I SHALL NOT BE FOUND"
	_, ok := licFromName(liclist, licName)
	if ok {
		t.Fatalf("License name '%s' should not have been found", licName)
	}

	// test character case deviation name identification
	licNameCorrect := "Creative Commons Attribution 4.0 International Public License"
	licName = " creative commons attribution 4.0 International Public License "
	lic, ok := licFromName(liclist, licName)
	if !ok {
		t.Fatalf("Error finding case deviant license name: '%s'", licName)
	}
	if lic.Name != licNameCorrect {
		t.Fatalf("Found invalid license: '%s' expected '%s'", lic.Name, licNameCorrect)
	}

	// test identification by alias
	licAlias := "  cc BY 4.0  "
	lic, ok = licFromName(liclist, licAlias)
	if !ok {
		t.Fatalf("Error finding license by alias: '%s'", licAlias)
	}
	if lic.Name != licNameCorrect {
		t.Fatalf("Found invalid license by alias: '%s' expected '%s'", lic.Name, licNameCorrect)
	}
}

func TestCleancompstr(t *testing.T) {
	instr := "  aLLcasEs  "
	expected := "allcases"
	outstr := cleancompstr(instr)
	if outstr != expected {
		t.Fatalf("Error string cleanup: '%s' expected: '%s'", outstr, expected)
	}
}

func TestLicenseWarnings(t *testing.T) {
	var warnings []string
	yada := &libgin.RepositoryYAML{
		License: &libgin.License{},
	}

	// Test all entries unknown, no license file access warnings
	checkwarn := licenseWarnings(yada, "", warnings)
	if len(checkwarn) != 3 {
		t.Fatalf("Unexpected warnings(%d): %v", len(checkwarn), checkwarn)
	}
	if !strings.Contains(checkwarn[0], "License URL (datacite) not found: ''") {
		t.Fatalf("Missing unknown license URL warning: %v", checkwarn)
	}
	if !strings.Contains(checkwarn[1], "License name (datacite) not found: ''") {
		t.Fatalf("Missing unknown license name warning: %v", checkwarn)
	}
	if !strings.Contains(checkwarn[2], "Could not access license file") {
		t.Fatalf("Missing failed license access warning: %v", checkwarn)
	}

	// Test all entries unknown, license file header unknown warnings
	// Use github gin-doi Makefile as invalid license header file
	licFileURL := "https://raw.githubusercontent.com/G-Node/gin-doi/master/Makefile"
	checkwarn = licenseWarnings(yada, licFileURL, warnings[:0])
	if len(checkwarn) != 3 {
		t.Fatalf("Unexpected warnings(%d): %v", len(checkwarn), checkwarn)
	}
	if !strings.Contains(checkwarn[2], "License file content header not found: '") {
		t.Fatalf("Missing unknown license file header warning: %v", checkwarn)
	}

	// Test all mismatch yURL!=yName!=fHeader
	// Uses GIN-DOI github library LICENSE (BSD3) as reference license file
	yada.License.URL = "https://creativecommons.org/publicdomain/zero/1.0"
	yada.License.Name = "MIT License"
	licFileURL = "https://raw.githubusercontent.com/G-Node/gin-doi/master/LICENSE"
	checkwarn = licenseWarnings(yada, licFileURL, warnings[:0])
	if len(checkwarn) != 2 {
		t.Fatalf("yURL!=yName!=File: unexpected warnings(%d): %v", len(checkwarn), checkwarn)
	}
	if !strings.Contains(checkwarn[0], "License URL/Name mismatch: 'CC0 1.0 Universal'/'The MIT License'") {
		t.Fatalf("Invalid yURL!=yName!=File warning: %v", checkwarn)
	}
	if !strings.Contains(checkwarn[1], "License name/file header mismatch: 'The MIT License'/'The 3-Clause BSD License'") {
		t.Fatalf("Invalid yURL!=yName!=File warning: %v", checkwarn)
	}

	// Test mismatch yURL!=(yName==fHeader)
	// Uses GIN-DOI github library LICENSE (BSD3) as reference license file
	yada.License.URL = "https://creativecommons.org/publicdomain/zero/1.0"
	yada.License.Name = "The 3-Clause BSD License"
	licFileURL = "https://raw.githubusercontent.com/G-Node/gin-doi/master/LICENSE"
	checkwarn = licenseWarnings(yada, licFileURL, warnings[:0])
	if len(checkwarn) != 1 {
		t.Fatalf("yURL!=yName==File: unexpected warnings(%d): %v", len(checkwarn), checkwarn)
	}
	if !strings.Contains(checkwarn[0], "License URL/Name mismatch: 'CC0 1.0 Universal'/'The 3-Clause BSD License'") {
		t.Fatalf("Invalid yURL!=yName==File warning: %v", checkwarn)
	}

	// Test mismatch (yURL==yName)!=fHeader
	// Uses GIN-DOI github library LICENSE (BSD3) as reference license file
	yada.License.URL = "https://opensource.org/licenses/MIT"
	yada.License.Name = "MIT License"
	licFileURL = "https://raw.githubusercontent.com/G-Node/gin-doi/master/LICENSE"
	checkwarn = licenseWarnings(yada, licFileURL, warnings[:0])
	if len(checkwarn) != 1 {
		t.Fatalf("yURL==yName!=File: unexpected warnings(%d): %v", len(checkwarn), checkwarn)
	}
	if !strings.Contains(checkwarn[0], "License name/file header mismatch: 'The MIT License'/'The 3-Clause BSD License'") {
		t.Fatalf("Invalid yURL==yName!=File warning: %v", checkwarn)
	}

	// Test URL, Name and Header match; uses GIN-DOI github library LICENSE (BSD3) as reference license file.
	yada.License.URL = "https://opensource.org/licenses/BSD-3-Clause"
	yada.License.Name = "BSD-3-Clause" // valid alias
	licFileURL = "https://raw.githubusercontent.com/G-Node/gin-doi/master/LICENSE"

	checkwarn = licenseWarnings(yada, licFileURL, warnings[:0])
	if len(checkwarn) > 0 {
		t.Fatalf("All match: unexpected warnings(%d): %v", len(checkwarn), checkwarn)
	}
}

func TestContentSizeWarning(t *testing.T) {
	warnings := []string{}
	md := &libgin.RepositoryMetadata{}

	// check nothing added on non-existing directory
	warnings = contentSizeWarning("/tmp/I/dont/exist", md, warnings)
	if len(warnings) != 0 {
		t.Fatalf("invalid dir: expected empty warning list but got: %q", warnings)
	}

	// check nothing added on non-git repository
	// check annex is available to the test; stop the test otherwise
	hasAnnex, err := annexAvailable()
	if err != nil {
		t.Fatalf("Error checking git annex: %q", err.Error())
	} else if !hasAnnex {
		t.Skipf("Annex is not available, skipping test...\n")
	}

	targetpath := t.TempDir()

	// test no warning on non-git dir
	warnings = contentSizeWarning(targetpath, md, warnings)
	if len(warnings) != 0 {
		t.Fatalf("non git: expected empty warning list but got: %q", warnings)
	}

	// initialize git directory
	stdout, stderr, err := remoteGitCMD(targetpath, false, "init")
	if err != nil {
		t.Fatalf("could not initialize git repo: %q, %q, %q", err.Error(), stdout, stderr)
	}
	// initialize annex
	stdout, stderr, err = remoteGitCMD(targetpath, true, "init")
	if err != nil {
		t.Fatalf("could not init annex: %q, %q, %q", err.Error(), stdout, stderr)
	}

	// test no error on no annex files
	warnings = contentSizeWarning(targetpath, md, warnings)
	if len(warnings) != 1 {
		t.Fatalf("no annex files: expected warning list entry but got: %q", warnings)
	} else if !strings.Contains(warnings[0], "n/a") || !strings.Contains(warnings[0], "Annex content size") {
		t.Fatalf("no annex files: expected warning list entry but got: %q", warnings)
	}
	warnings = []string{}

	// create annex data file
	fname := "datafile.txt"
	fpath := filepath.Join(targetpath, fname)
	err = ioutil.WriteFile(fpath, []byte("some data"), 0777)
	if err != nil {
		t.Fatalf("Error creating annex data file %q", err.Error())
	}
	// add file to the annex; note that this will also lock the file by annex default
	stdout, stderr, err = remoteGitCMD(targetpath, true, "add", fpath)
	if err != nil {
		t.Fatalf("error on git annex add file\n%s\n%s\n%s", err.Error(), stdout, stderr)
	}
	// uninit annex file so the cleanup can happen but ignore any further issues
	// the temp folder will get cleaned up eventually anyway.
	defer remoteGitCMD(targetpath, true, "uninit", fpath)

	// check warning but no zipsize on empty metadata
	warnings = contentSizeWarning(targetpath, md, warnings)
	if len(warnings) != 1 {
		t.Fatalf("empty metadata: expected annex size warning but got: %s", warnings)
	} else if !strings.Contains(warnings[0], "n/a") || !strings.Contains(warnings[0], "Annex content size") {
		t.Fatalf("empty metadata: expected empty zip size but got: %s", warnings)
	}

	// check warning on empty metadata sizes
	warnings = []string{}
	md.DataCite = &libgin.DataCite{}
	warnings = contentSizeWarning(targetpath, md, warnings)
	if len(warnings) != 1 {
		t.Fatalf("empty metadata size: expected an annex size warning but got: %s", warnings)
	}
	if !strings.Contains(warnings[0], "n/a") || !strings.Contains(warnings[0], "Annex content size") {
		t.Fatalf("empty metadata size: expected empty zip size but got: %s", warnings)
	}

	// check zip size and warning append
	zipsize := "zipsize12"
	md.DataCite.Sizes = &[]string{zipsize}
	warnings = contentSizeWarning(targetpath, md, warnings)
	if len(warnings) != 2 {
		t.Fatalf("valid entry: unexpected number of warnings: %q", warnings)
	}
	if !strings.Contains(warnings[1], zipsize) {
		t.Fatalf("valid entry: missing zip size entry: %q", warnings[1])
	}
}

func TestAuthorWarnings(t *testing.T) {
	var warnings []string
	yada := &libgin.RepositoryYAML{}

	// Check no author warning on empty struct or empty Author
	checkwarn := authorWarnings(yada, warnings)
	if len(checkwarn) != 0 {
		t.Fatalf("Invalid number of messages(%d): %v", len(checkwarn), checkwarn)
	}

	var auth []libgin.Author
	auth = append(auth, libgin.Author{})
	yada.Authors = auth

	checkwarn = authorWarnings(yada, warnings)
	if len(checkwarn) != 0 {
		t.Fatalf("Invalid number of messages(%d): %v", len(checkwarn), checkwarn)
	}

	// Check warning on non identifiable ID that looks like an ORCID
	yada.Authors[0].ID = "0000-0000-0000-0000"
	checkwarn = authorWarnings(yada, warnings)
	if len(checkwarn) != 1 {
		t.Fatalf("Invalid number of messages(%d): %v", len(checkwarn), checkwarn)
	}
	if !strings.Contains(checkwarn[0], "has an ORCID-like unspecified ID") {
		t.Fatalf("Expected ORCID like ID message: %v", checkwarn[0])
	}

	// Check warning on non identifiable ID
	yada.Authors[0].ID = "I:amNoID"
	checkwarn = authorWarnings(yada, warnings)
	if len(checkwarn) != 1 {
		t.Fatalf("Invalid number of messages(%d): %v", len(checkwarn), checkwarn)
	}
	if !strings.Contains(checkwarn[0], "has an unknown ID") {
		t.Fatalf("Expected unknown ID message: %v", checkwarn[0])
	}

	// Check warning on known ID type, but missing value (orcid and researcherid)
	yada.Authors[0].ID = "orCid:"
	checkwarn = authorWarnings(yada, warnings)
	if len(checkwarn) != 1 {
		t.Fatalf("Invalid number of messages(%d): %v", len(checkwarn), checkwarn)
	}
	if !strings.Contains(checkwarn[0], "has an empty ID value") {
		t.Fatalf("Expected empty ORCID value message: %v", checkwarn[0])
	}

	yada.Authors[0].ID = "researcherID:"
	checkwarn = authorWarnings(yada, warnings)
	if len(checkwarn) != 1 {
		t.Fatalf("Invalid number of messages(%d): %v", len(checkwarn), checkwarn)
	}
	if !strings.Contains(checkwarn[0], "has an empty ID value") {
		t.Fatalf("Expected empty researcherid value message: %v", checkwarn[0])
	}

	// Check warning on duplicate ORCID, researchID
	yada.Authors[0].ID = "orcid:0000-0001-6744-1159"
	auth = yada.Authors
	auth = append(auth, libgin.Author{ID: "researcherid:k-3714-2014"})
	auth = append(auth, libgin.Author{ID: "ORCID:0000-0001-6744-1159"})
	auth = append(auth, libgin.Author{ID: "researcherID:K-3714-2014"})
	yada.Authors = auth

	checkwarn = authorWarnings(yada, warnings)
	if len(checkwarn) != 2 {
		t.Fatalf("Invalid number of messages(%d): %v", len(checkwarn), checkwarn)
	}
	for _, warn := range checkwarn {
		if !strings.Contains(warn, "have the same ID:") {
			t.Fatalf("Expected duplicate ID message: %v", warn)
		}
	}

	// Check no warning on valid entries
	var cleanauth []libgin.Author
	cleanauth = append(cleanauth, libgin.Author{ID: "researcherid:k-3714-2014"})
	cleanauth = append(cleanauth, libgin.Author{ID: "ORCID:0000-0001-6744-1159"})
	yada.Authors = cleanauth
	checkwarn = authorWarnings(yada, warnings)
	if len(checkwarn) != 0 {
		t.Fatalf("Invalid number of messages(%d): %v", len(checkwarn), checkwarn)
	}
}

func TestAuthorIDWarnings(t *testing.T) {
	var warnings []string
	yada := &libgin.RepositoryYAML{}

	// Check no author warning on empty struct or empty Author
	checkwarn := authorIDWarnings(yada, warnings)
	if len(checkwarn) != 0 {
		t.Fatalf("Invalid number of messages(%d): %v", len(checkwarn), checkwarn)
	}

	var auth []libgin.Author
	auth = append(auth, libgin.Author{})
	yada.Authors = auth

	checkwarn = authorIDWarnings(yada, warnings)
	if len(checkwarn) != 0 {
		t.Fatalf("Invalid number of messages(%d): %v", len(checkwarn), checkwarn)
	}

	// Check warning on false author ID entries
	yada.Authors[0].ID = "orcid:x000-0001-2345-6789"
	checkwarn = authorIDWarnings(yada, warnings)
	if len(checkwarn) != 1 {
		t.Fatalf("Invalid number of messages(%d): %v", len(checkwarn), checkwarn)
	}
	if !strings.Contains(checkwarn[0], "ID was not found at the ID service") {
		t.Fatalf("Expected not found ID message: %v", checkwarn[0])
	}
}

func TestValidateDataCiteValues(t *testing.T) {
	invResource := "<strong>ResourceType</strong> must be one of the following:"
	invReference := "Reference type (<strong>RefType</strong>) must be one of the following:"

	// Check required resourceType message on empty struct
	yada := &libgin.RepositoryYAML{}

	msgs := validateDataCiteValues(yada)
	if len(msgs) != 1 {
		t.Fatalf("Invalid number of messages(%d): %v", len(msgs), msgs)
	}
	if !strings.Contains(msgs[0], invResource) {
		t.Fatalf("Expected resource type message: %v", msgs[0])
	}

	// Check invalid resource type
	yada.ResourceType = "Idonotexist"
	msgs = validateDataCiteValues(yada)
	if len(msgs) != 1 {
		t.Fatalf("Invalid number of messages(%d): %v", len(msgs), msgs)
	}
	if !strings.Contains(msgs[0], invResource) {
		t.Fatalf("Expected resource message: %v", msgs[0])
	}

	// Check fail on an existing but empty reference
	yada.ResourceType = "Dataset"
	var ref []libgin.Reference
	ref = append(ref, libgin.Reference{})
	yada.References = ref

	msgs = validateDataCiteValues(yada)
	if len(msgs) != 1 {
		t.Fatalf("Invalid number of messages(%d): %v", len(msgs), msgs)
	}
	if !strings.Contains(msgs[0], invReference) {
		t.Fatalf("Expected reference message: %v", msgs[0])
	}

	// Check fail on invalid reference
	yada.References[0].RefType = "idonotexist"
	msgs = validateDataCiteValues(yada)
	if len(msgs) != 1 {
		t.Fatalf("Invalid number of messages(%d): %v", len(msgs), msgs)
	}
	if !strings.Contains(msgs[0], invReference) {
		t.Fatalf("Expected reference message: %v", msgs[0])
	}

	// Check all valid
	yada.References[0].RefType = "IsSupplementTo"

	msgs = validateDataCiteValues(yada)
	if len(msgs) != 0 {
		t.Fatalf("Unexpected messages: %v", msgs)
	}
}

func TestReferenceWarnings(t *testing.T) {
	var warnings []string
	// Check warnings on empty struct
	yada := &libgin.RepositoryYAML{}

	warn := referenceWarnings(yada, warnings)
	if len(warn) != 0 {
		t.Fatalf("Invalid number of messages(%d): %v", len(warn), warn)
	}

	// Check refIDType warning on empty ID
	var ref []libgin.Reference
	ref = append(ref, libgin.Reference{RefType: "IsSupplementTo"})
	yada.References = ref
	warn = referenceWarnings(yada, warnings)
	if len(warn) != 1 {
		t.Fatalf("Invalid number of messages(%d): %v", len(warn), warn)
	}
	if !strings.Contains(warn[0], "has no related ID type:") {
		t.Fatalf("Unexpected related ID type warning: %v", warn)
	}

	// Check refIDType warning on missing id type
	yada.References[0].ID = "10.12751/g-node.6953bb"
	warn = referenceWarnings(yada, warnings)
	if len(warn) != 1 {
		t.Fatalf("Invalid number of messages(%d): %v", len(warn), warn)
	}
	if !strings.Contains(warn[0], "has no related ID type:") {
		t.Fatalf("Unexpected related ID type warning: %v", warn)
	}

	// Check warning on "name" field used and warning on uncommon referenceType
	yada.References[0].ID = "doi:10.12751/g-node.6953bb"
	yada.References[0].Name = "used instead of citation"
	yada.References[0].RefType = "IsDescribedBy"

	warn = referenceWarnings(yada, warnings)
	if len(warn) != 2 {
		t.Fatalf("Invalid number of messages(%d): %v", len(warn), warn)
	}
	if !strings.Contains(warn[0], "uses old 'Name' field instead of 'Citation'") {
		t.Fatalf("Unexpected name field warning: %v", warn)
	}
	if !strings.Contains(warn[1], " uses refType 'IsDescribedBy'") {
		t.Fatalf("Unexpected reference type warning: %v", warn)
	}

	// Check no warnings on valid reference
	yada.References[0].Name = ""
	yada.References[0].Citation = "validcitation"
	yada.References[0].RefType = "IsSupplementTo"
	yada.References[0].ID = "doi:10.12751/g-node.6953bb"
	warn = referenceWarnings(yada, warnings)
	if len(warn) != 0 {
		t.Fatalf("Invalid number of messages(%d): %v", len(warn), warn)
	}
}
