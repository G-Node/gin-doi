package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
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

func TestCleanupcompstr(t *testing.T) {
	instr := "  aLLcasEs  "
	expected := "allcases"
	outstr := cleancompstr(instr)
	if outstr != expected {
		t.Fatalf("Error string cleanup: '%s' expected: '%s'", outstr, expected)
	}
}
