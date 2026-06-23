// Package frontend embeds the Pizza Tracker SPA assets so the backend binary
// can serve them without a separate frontend directory at runtime.
package frontend

import (
	"embed"
	"io/fs"
)

// app.css is the Tailwind CSS output compiled ahead of time from
// frontend/input.css and served alongside index.html at /app.css.
//
//go:embed index.html app.css
var assets embed.FS

// Assets is the embedded SPA file system (index.html plus the compiled
// app.css stylesheet), served by the dashboard server via http.FileServerFS.
var Assets fs.FS = assets
