package core

import (
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// PageHandler is the catch-all route handler, for serving static web pages.
func (b *Blog) PageHandler(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path
	log.Debug("Catch-all page handler invoked for request URI: %s", path)

	// Remove trailing slashes by redirecting them away.
	if len(path) > 1 && path[len(path)-1] == '/' {
		b.Redirect(w, strings.TrimRight(path, "/"))
		return
	}

	// Restrict special paths.
	if strings.HasPrefix(strings.ToLower(path), "/.") {
		b.Forbidden(w, r)
		return
	}

	// Search for a file that matches their URL.
	filepath, err := b.ResolvePath(path)
	if err != nil {
		b.NotFound(w, r, "The page you were looking for was not found.")
		return
	}

	// Is it a template file?
	if strings.HasSuffix(filepath.URI, ".gohtml") || strings.HasSuffix(filepath.URI, ".html") {
		b.RenderTemplate(w, r, filepath.URI, nil)
		return
	}

	http.ServeFile(w, r, filepath.Absolute)
}

// Filepath represents a file discovered in the document roots, and maintains
// both its relative and absolute components.
type Filepath struct {
	// Canonicalized URI version of the file resolved on disk,
	// possible with a file extension injected.
	// (i.e. "/about" -> "about.html")
	URI      string
	Relative string // Relative path including document root (i.e. "root/about.html")
	Absolute string // Absolute path on disk (i.e. "/opt/blog/root/about.html")
}

func (f Filepath) String() string {
	return f.Relative
}

// ResolvePath matches a filesystem path to a relative request URI.
//
// This checks the UserRoot first and then the DocumentRoot. This way the user
// may override templates from the core app's document root.
func (b *Blog) ResolvePath(path string) (Filepath, error) {
	// Strip leading slashes.
	if path[0] == '/' {
		path = strings.TrimPrefix(path, "/")
	}

	// If you need to debug this function, edit this block.
	debug := func(tmpl string, args ...interface{}) {
		if false {
			log.Debug(tmpl, args...)
		}
	}

	debug("Resolving filepath for URI: %s", path)
	for _, root := range []string{b.DocumentRoot, b.UserRoot} {
		if len(root) == 0 {
			continue
		}

		// Resolve the file path.
		relPath := filepath.Join(root, path)
		absPath, err := filepath.Abs(relPath)
		if err != nil {
			log.Error("%v", err)
		}

		debug("Expected filepath: %s", absPath)

		// Found an exact hit?
		if stat, err := os.Stat(absPath); !os.IsNotExist(err) && !stat.IsDir() {
			debug("Exact filepath found: %s", absPath)
			return Filepath{path, relPath, absPath}, nil
		}

		// Try some supported suffixes.
		suffixes := []string{
			".gohtml",
			".html",
			"/index.gohtml",
			"/index.html",
			".md",
			"/index.md",
		}
		for _, suffix := range suffixes {
			test := absPath + suffix
			if stat, err := os.Stat(test); !os.IsNotExist(err) && !stat.IsDir() {
				debug("Filepath found via suffix %s: %s", suffix, test)
				return Filepath{path + suffix, relPath + suffix, test}, nil
			}
		}
	}

	return Filepath{}, errors.New("not found")
}
