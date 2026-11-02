// Package appicon serves the icons macOS shows for App bundles through a custom URL scheme.
package appicon

import (
	"net/http"
	"path/filepath"
	"strings"
	"sync"
)

// Scheme is registered with mygo.Protocol; pages load appicon://localhost/?path=/Applications/Foo.app,
// or appicon://localhost/type?ext=pdf (or ?folder=1) for the icon of a file type.
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
		if r.URL.Path == "/type" {
			// appicon://localhost/type?ext=xlsx, or ?folder=1: one icon per type, so the cache stays small.
			ext, folder := strings.ToLower(r.URL.Query().Get("ext")), r.URL.Query().Has("folder")
			if len(ext) > 16 || strings.ContainsFunc(ext, func(c rune) bool { return !('a' <= c && c <= 'z' || '0' <= c && c <= '9') }) {
				http.Error(w, "an alphanumeric extension is required", http.StatusBadRequest)
				return
			}
			key := "type:" + ext
			if folder {
				key = "folder"
			}
			png, ok := cache.Load(key)
			if !ok {
				data, err := renderType(ext, folder, pixels)
				if err != nil {
					http.Error(w, err.Error(), http.StatusNotFound)
					return
				}
				png, _ = cache.LoadOrStore(key, data)
			}
			w.Header().Set("Content-Type", "image/png")
			w.Header().Set("Cache-Control", "max-age=86400")
			w.Write(png.([]byte))
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
