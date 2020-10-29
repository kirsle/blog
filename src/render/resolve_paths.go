package render

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/kirsle/blog/src/log"
	"github.com/kirsle/blog/src/root"
)

// Blog configuration bindings.
var (
	UserRoot     *string
	DocumentRoot *string
)

// File extensions and URL suffixes that map to real files on disk, but which
// have suffixes hidden from the URL.
var hiddenSuffixes = []string{
	".gohtml",
	".html",
	"/index.gohtml",
	"/index.html",
	".md",
	"/index.md",
}

// Filepath represents a file discovered in the document roots, and maintains
// both its relative and absolute components.
type Filepath struct {
	// Canonicalized URI version of the file resolved on disk,
	// possible with a file extension injected.
	// (i.e. "/about" -> "about.html")
	URI      string
	Basename string
	Relative string // Relative path including document root (i.e. "root/about.html")
	Absolute string // Absolute path on disk (i.e. "/opt/blog/root/about.html")

	// If file was resolved to embedded bindata, this is the bindata key name.
	// Zero value means it resolved to a file on filesystem.
	BindataKey string
}

func (f Filepath) String() string {
	return f.Relative
}

// ResolvePath matches a filesystem path to a relative request URI.
//
// This checks the UserRoot first and then the DocumentRoot. This way the user
// may override templates from the core app's document root.
func ResolvePath(path string) (Filepath, error) {
	// Strip leading slashes.
	if path[0] == '/' {
		path = strings.TrimPrefix(path, "/")
	}

	// If you need to debug this function, edit this block.
	debug := func(tmpl string, args ...interface{}) {
		if true { // edit this to enable
			log.Debug(tmpl, args...)
		}
	}

	debug("ResolvePath(%s) called", path)

	if len(*UserRoot) > 0 {
		debug("1. Resolving filepath for URI in user root: %s", path)

		// Resolve the file path.
		relPath := filepath.Join(*UserRoot, path)
		absPath, err := filepath.Abs(relPath)
		basename := filepath.Base(relPath)
		if err != nil {
			log.Error("%v", err)
		}

		debug("   Expected filepath: %s", absPath)

		// Found an exact hit?
		if stat, err := os.Stat(absPath); !os.IsNotExist(err) && !stat.IsDir() {
			debug("   + Exact filepath found: %s", absPath)
			return Filepath{
				URI:      path,
				Basename: basename,
				Relative: relPath,
				Absolute: absPath,
			}, nil
		}

		// Try some supported suffixes.
		for _, suffix := range hiddenSuffixes {
			test := absPath + suffix
			if stat, err := os.Stat(test); !os.IsNotExist(err) && !stat.IsDir() {
				debug("   + Filepath found via suffix %s: %s", suffix, test)
				return Filepath{
					URI:      path + suffix,
					Basename: basename + suffix,
					Relative: relPath + suffix,
					Absolute: test,
				}, nil
			}
		}
	}

	debug("2. Not found in filesystem, checking bindata for: %s", path)
	{
		// Exact hit?
		if _, err := root.Asset(path); err == nil {
			debug("   Found in bindata as: %s", path)
			return Filepath{
				URI:        path,
				Basename:   filepath.Base(path),
				Relative:   path,
				Absolute:   path,
				BindataKey: path,
			}, nil
		}

		// Try some supported suffixes.
		for _, suffix := range hiddenSuffixes {
			test := path + suffix
			if _, err := root.Asset(test); err == nil {
				debug("   Filepath found via suffix %s: %s", suffix, test)
				return Filepath{
					URI:        test,
					Basename:   filepath.Base(test),
					Relative:   test,
					Absolute:   test,
					BindataKey: test,
				}, nil
			}
		}
	}

	return Filepath{}, errors.New("not found")
}

// HasHTMLSuffix returns whether the file path will be renderable as HTML
// for the front-end. Basically, whether it ends with a .gohtml, .html or .md
// suffix and/or is an index page.
func HasHTMLSuffix(path string) bool {
	for _, suffix := range hiddenSuffixes {
		if strings.HasSuffix(path, suffix) {
			return true
		}
	}
	return false
}

// URLFromPath returns an HTTP path that matches the file path on disk.
//
// For example, given the file path "folder/page.md" it would return the string
// "/folder/page"
func URLFromPath(path string) string {
	// Strip leading slashes.
	if path[0] == '/' {
		path = strings.TrimPrefix(path, "/")
	}

	// Hide-able suffixes.
	for _, suffix := range hiddenSuffixes {
		if strings.HasSuffix(path, suffix) {
			path = strings.TrimSuffix(path, suffix)
			break
		}
	}

	return fmt.Sprintf("/%s", path)
}
