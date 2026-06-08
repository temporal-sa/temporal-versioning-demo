// Package frontend embeds the Pizza Tracker SPA assets so the backend binary
// can serve them without a separate frontend directory at runtime.
package frontend

import (
	"embed"
	"io/fs"
)

//go:embed index.html
var assets embed.FS

// Assets is the embedded SPA file system, served by the dashboard server via
// http.FileServerFS.
var Assets fs.FS = assets
