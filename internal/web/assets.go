package web

import (
	"embed"
	"io/fs"
)

//go:embed all:dist
var frontend embed.FS

// Assets returns the built frontend, or just the placeholder in API-only development.
func Assets() (fs.FS, error) {
	return fs.Sub(frontend, "dist")
}
