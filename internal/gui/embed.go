package gui

import (
	"embed"
	"io/fs"
	"net/http"
)

//go:embed web/*
var embeddedWebFS embed.FS

// GetFileSystem returns the HTTP file system for the embedded web assets
func GetFileSystem() (http.FileSystem, error) {
	sub, err := fs.Sub(embeddedWebFS, "web")
	if err != nil {
		return nil, err
	}
	return http.FS(sub), nil
}
