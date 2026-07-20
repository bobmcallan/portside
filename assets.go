package main

import (
	"embed"
	"net/http"
	"strconv"
)

//go:embed assets/fonts/space-grotesk-latin.woff2
var assetFS embed.FS

const (
	fontURLPath   = "/assets/fonts/space-grotesk-latin.woff2"
	fontEmbedPath = "assets/fonts/space-grotesk-latin.woff2"
)

// registerAssets mounts embedded static assets (currently the self-hosted
// Space Grotesk latin subset) on mux.
func registerAssets(mux *http.ServeMux) {
	mux.HandleFunc(fontURLPath, serveFont)
}

func serveFont(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if r.URL.Path != fontURLPath {
		http.NotFound(w, r)
		return
	}
	b, err := assetFS.ReadFile(fontEmbedPath)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "font/woff2")
	w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
	w.Header().Set("Content-Length", strconv.Itoa(len(b)))
	if r.Method == http.MethodHead {
		w.WriteHeader(http.StatusOK)
		return
	}
	_, _ = w.Write(b)
}
