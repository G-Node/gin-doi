package main

import (
	"net/http"
)

type AssetFS struct {
	fs http.FileSystem
}

func NewAssetFS(path string) AssetFS {
	dir := http.Dir(path)
	return AssetFS{dir}
}

// Open files but not directories. Disallows directory listing of asset directories.
func (as AssetFS) Open(path string) (http.File, error) {
	fp, err := as.fs.Open(path)
	if err != nil {
		return nil, err
	}

	stat, err := fp.Stat()
	if err != nil {
		return nil, err
	}

	if stat.IsDir() {
		return nil, err
	}

	return fp, nil
}
