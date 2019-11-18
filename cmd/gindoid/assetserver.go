package main

import (
	"net/http"
)

// AssetFS embeds a http.FileSystem for serving static assets.
type AssetFS struct {
	fs http.FileSystem
}

// newAssetFS creates a new asset filesystem server rooted at the given path.
func newAssetFS(path string) AssetFS {
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
