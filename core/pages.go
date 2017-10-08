package core

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// PageHandler is the catch-all route handler, for serving static web pages.
func (b *Blog) PageHandler(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path

	// Remove trailing slashes by redirecting them away.
	if len(path) > 1 && path[len(path)-1] == '/' {
		Redirect(w, strings.TrimRight(path, "/"))
		return
	}

	// Search for a file that matches their URL.
	log.Debug("Resolving filepath for URI: %s", path)
	for _, root := range []string{b.DocumentRoot, b.UserRoot} {
		relPath := filepath.Join(root, path)
		absPath, err := filepath.Abs(relPath)
		if err != nil {
			log.Error("%v", err)
		}

		log.Debug("Expected filepath: %s", absPath)

		// Found an exact hit?
		if stat, err := os.Stat(absPath); !os.IsNotExist(err) && !stat.IsDir() {
			log.Debug("Exact filepath found: %s", absPath)
			http.ServeFile(w, r, absPath)
			return
		}

		// Try some supported suffixes.
		suffixes := []string{
			".html",
			"/index.html",
			".md",
			"/index.md",
		}
		for _, suffix := range suffixes {
			if stat, err := os.Stat(absPath + suffix); !os.IsNotExist(err) && !stat.IsDir() {
				log.Debug("Filepath found via suffix %s: %s", suffix, absPath+suffix)
				http.ServeFile(w, r, absPath+suffix)
				return
			}
		}
	}

	// No file, must be a 404.
	http.NotFound(w, r)
}
