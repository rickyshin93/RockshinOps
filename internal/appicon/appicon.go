// Package appicon serves the icons macOS shows for App bundles through a custom URL scheme.
package appicon

import (
	"net/http"
	"path/filepath"
	"strings"
	"sync"
)

// Scheme is registered with mygo.Protocol; pages load appicon://localhost/?path=/Applications/Foo.app.
const Scheme = "appicon"

// pixels covers a 32-point icon on a 2x display.
const pixels = 64

var cache sync.Map

func Handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/computer" {
			data, err := renderComputer()
			if err != nil {
				http.Error(w, err.Error(), http.StatusNotFound)
				return
			}
			w.Header().Set("Content-Type", "image/png")
			w.Header().Set("Cache-Control", "max-age=86400")
			w.Write(data)
			return
		}
		path := r.URL.Query().Get("path")
		if !filepath.IsAbs(path) || filepath.Clean(path) != path || !strings.HasSuffix(path, ".app") {
			http.Error(w, "an absolute .app path is required", http.StatusBadRequest)
			return
		}
		png, ok := cache.Load(path)
		if !ok {
			data, err := render(path, pixels)
			if err != nil {
				http.Error(w, err.Error(), http.StatusNotFound)
				return
			}
			png, _ = cache.LoadOrStore(path, data)
		}
		w.Header().Set("Content-Type", "image/png")
		w.Header().Set("Cache-Control", "max-age=86400")
		w.Write(png.([]byte))
	})
}
