package main

import (
	"archive/zip"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
)

func TestMakeZip(t *testing.T) {
	targetpath, err := ioutil.TempDir("", "test_libgin_makezip")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(targetpath)

	ziproot := filepath.Join(targetpath, "mkzip")

	// Create test directory tree
	handletestdir := func() error {
		// Create directories
		inclpath := filepath.Join(ziproot, "included")
		exclpath := filepath.Join(ziproot, ".excluded")
		gitpath := filepath.Join(ziproot, ".git")
		if err = os.MkdirAll(inclpath, 0755); err != nil {
			return fmt.Errorf("Error creating directory %s: %v", inclpath, err)
		}
		if err = os.MkdirAll(exclpath, 0755); err != nil {
			return fmt.Errorf("Error creating directory %s: %v", exclpath, err)
		}
		if err = os.MkdirAll(gitpath, 0755); err != nil {
			return fmt.Errorf("Error creating directory %s: %v", gitpath, err)
		}

		mkfile := func(currfile string) error {
			fp, err := os.Create(currfile)
			if err != nil {
				return fmt.Errorf("Error creating file %s: %v", currfile, err)
			}
			defer fp.Close()
			return nil
		}
		// Create files
		if err = mkfile(filepath.Join(ziproot, "included.md")); err != nil {
			return err
		}
		if err = mkfile(filepath.Join(gitpath, "excluded.md")); err != nil {
			return err
		}
		if err = mkfile(filepath.Join(inclpath, "included.md")); err != nil {
			return err
		}
		if err = mkfile(filepath.Join(inclpath, "not_excluded.md")); err != nil {
			return err
		}
		if err = mkfile(filepath.Join(exclpath, "excluded.md")); err != nil {
			return err
		}

		return nil
	}
	if err = handletestdir(); err != nil {
		t.Fatalf("%v", err)
	}

	zipbasename := "test_makezip.zip"
	zipfilename := filepath.Join(targetpath, zipbasename)

	// Checks that files in directories ".git" and ".excluded" are excluded and
	// file "not_excluded" is still added.
	exclude := []string{".git", ".excluded", "not_excluded.md"}

	handlezip := func() error {
		fn := fmt.Sprintf("zip(%s, %s)", ziproot, zipfilename)
		source, err := filepath.Abs(ziproot)
		if err != nil {
			return fmt.Errorf("Failed to get abs path for source directory in function '%s': %v", fn, err)
		}

		zipfilename, err = filepath.Abs(zipfilename)
		if err != nil {
			return fmt.Errorf("Failed to get abs path for target zip file in function '%s': %v", fn, err)
		}

		zipfp, err := os.Create(zipfilename)
		if err != nil {
			return fmt.Errorf("Failed to create zip file for writing in function '%s': %v", fn, err)
		}
		defer zipfp.Close()

		origdir, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("Failed to get working directory in function '%s': %v", fn, err)
		}
		defer os.Chdir(origdir)

		if err := os.Chdir(source); err != nil {
			return fmt.Errorf("Failed to change to source directory to make zip file in function '%s': %v", fn, err)
		}

		if err := MakeZip(zipfp, exclude, "."); err != nil {
			return fmt.Errorf("Failed to change to source directory to make zip file in function '%s': %v", fn, err)
		}
		return nil
	}

	if err = handlezip(); err != nil {
		t.Fatalf("%v", err)
	}

	// Files included in the zip file
	incl := map[string]struct{}{
		"included/included.md":     {},
		"included/not_excluded.md": {},
		"included.md":              {},
	}
	// Files excluded from the zip file
	excl := map[string]struct{}{
		".git/excluded.md":      {},
		".excluded/excluded.md": {},
	}

	zipreader, err := zip.OpenReader(zipfilename)
	if err != nil {
		t.Fatalf("Error opening zip file: %v", err)
	}
	defer zipreader.Close()

	var includedCounter []string
	for _, file := range zipreader.File {
		if _, included := incl[file.Name]; !included {
			if _, notExcluded := excl[file.Name]; notExcluded {
				t.Fatalf("Not excluded file found: %s", file.Name)
			}
		} else {
			includedCounter = append(includedCounter, file.Name)
		}
	}
	if len(includedCounter) != len(incl) {
		t.Fatalf("Zip does not include correct number of elements: %v/%v\n%v", len(includedCounter), len(incl), includedCounter)
	}
}

func TestReadRepoYAML(t *testing.T) {
	invalid := "<xml>I am not a yaml file</xml>"
	_, err := readRepoYAML([]byte(invalid))
	if err == nil {
		t.Fatalf("Expected YAML read error")
	}
	valid := "key: value"
	_, err = readRepoYAML([]byte(valid))
	if err != nil {
		t.Fatalf("Could not read YAML")
	}
}
