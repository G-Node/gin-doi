package main

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

func Zip(src string, writers ...io.Writer) error {

	// ensure the src actually exists before trying to tar it
	if _, err := os.Stat(src); err != nil {
		return fmt.Errorf("Unable to zip files: %s", err.Error())
	}

	mw := io.MultiWriter(writers...)

	tw := zip.NewWriter(mw)
	defer tw.Close()

	// walk path
	return filepath.Walk(src, func(file string, fi os.FileInfo, err error) error {

		// return on any error
		if err != nil {
			return err
		}

		// create a new dir/file header
		header, err := zip.FileInfoHeader(fi)
		if err != nil {
			return err
		}
		// update the name to correctly reflect the desired destination when untaring
		header.Name = strings.TrimPrefix(strings.Replace(file, src, "", -1), string(filepath.Separator))

		// write the header
		w, err := tw.CreateHeader(header)
		if err != nil {
			return err
		}

		// return on directories since there will be no content to tar
		if fi.Mode().IsDir() {
			return nil
		}
		mode := fi.Mode()
		fmt.Print(mode)
		if fi.Mode()&os.ModeSymlink != 0 {
			data, err := os.Readlink(file)
			if err != nil {
				return err
			}
			if _, err := io.Copy(w, strings.NewReader(data)); err != nil {
				return err
			}
			return nil
		}

		// open files for taring
		f, err := os.Open(file)
		defer f.Close()
		if err != nil {
			return err
		}

		// copy file data into tar writer
		if _, err := io.Copy(w, f); err != nil {
			return err
		}

		return nil
	})
}
