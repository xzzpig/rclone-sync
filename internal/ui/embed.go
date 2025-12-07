package ui

import (
	"embed"
	"io/fs"
	"net/http"
)

//go:generate sh -c "cd ../../web && pnpm install && pnpm build"
//go:embed dist/*
var distFS embed.FS

// GetFileSystem returns the embedded filesystem for the frontend
func GetFileSystem() (http.FileSystem, error) {
	fsys, err := fs.Sub(distFS, "dist")
	if err != nil {
		return nil, err
	}
	return http.FS(fsys), nil
}
