package main

import (
	"embed"
	"io/fs"
)

//go:embed public/*
var uiFS embed.FS

func getUIFS() fs.FS {
	f, err := fs.Sub(uiFS, "public")
	if err != nil {
		return uiFS
	}
	return f
}
