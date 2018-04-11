package admin

import (
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/kirsle/blog/internal/render"
	"github.com/kirsle/blog/internal/responses"
)

// FileTree holds information about files in the document roots.
type FileTree struct {
	UserRoot bool // false = CoreRoot
	Files    []render.Filepath
}

func editorHandler(w http.ResponseWriter, r *http.Request) {
	// Editing a page?
	file := strings.Trim(r.FormValue("file"), "/")
	if len(file) > 0 {
		var (
			fp       string
			fromCore = r.FormValue("from") == "core"
			saving   = r.FormValue("action") == "save"
			deleting = r.FormValue("action") == "delete"
			body     = []byte{}
		)

		// Are they saving?
		if saving {
			fp = filepath.Join(*render.UserRoot, file)
			body = []byte(strings.Replace(r.FormValue("body"), "\r\n", "\n", -1))

			// Ensure the folders exist.
			dir, _ := filepath.Split(fp)
			err := os.MkdirAll(dir, 0755)
			if err != nil {
				responses.Flash(w, r, "Error saving: can't create folder %s: %s", dir, err)
			}

			// Write the file.
			err = ioutil.WriteFile(fp, body, 0644)
			if err != nil {
				responses.Flash(w, r, "Error saving: %s", err)
			} else {
				if render.HasHTMLSuffix(file) {
					responses.FlashAndRedirect(w, r, render.URLFromPath(file), "Page saved successfully!")
				} else {
					responses.FlashAndRedirect(w, r, "/admin/editor?file="+url.QueryEscape(file), "Page saved successfully!")
				}
				return
			}
		} else if deleting {
			fp = filepath.Join(*render.UserRoot, file)
			err := os.Remove(fp)
			if err != nil {
				responses.FlashAndRedirect(w, r, "/admin/editor", "Error deleting: %s", err)
			} else {
				responses.FlashAndRedirect(w, r, "/admin/editor", "Page deleted!")
				return
			}
		} else {
			// Where is the file from?
			if fromCore {
				fp = filepath.Join(*render.DocumentRoot, file)
			} else {
				fp = filepath.Join(*render.UserRoot, file)
			}

			// Check the file. If not found, check from the core root.
			f, err := os.Stat(fp)
			if os.IsNotExist(err) {
				fp = filepath.Join(*render.DocumentRoot, file)
				f, err = os.Stat(fp)
				if !os.IsNotExist(err) {
					// The file was found in the core.
					fromCore = true
				}
			}

			// If it exists, load it.
			if !os.IsNotExist(err) && !f.IsDir() {
				body, err = ioutil.ReadFile(fp)
				if err != nil {
					responses.Flash(w, r, "Error reading %s: %s", fp, err)
				}
			}

			// Default HTML boilerplate for .gohtml templates.
			if len(body) == 0 && strings.HasSuffix(fp, ".gohtml") {
				body = []byte("{{ define \"title\" }}Untitled Page{{ end }}\n" +
					"{{ define \"content\" }}\n<h1>Untitled Page</h1>\n\n{{ end }}")
			}
		}

		v := map[string]interface{}{
			"File":     file,
			"Path":     fp,
			"Body":     string(body),
			"FromCore": fromCore,
		}
		render.Template(w, r, "admin/editor", v)
		return
	}

	// Otherwise listing the index view.
	editorFileList(w, r)
}

// editorFileList handles the index view of /admin/editor.
func editorFileList(w http.ResponseWriter, r *http.Request) {
	// Listing the file tree?
	trees := []FileTree{}
	for i, root := range []string{*render.UserRoot, *render.DocumentRoot} {
		tree := FileTree{
			UserRoot: i == 0,
			Files:    []render.Filepath{},
		}

		filepath.Walk(root, func(path string, f os.FileInfo, err error) error {
			abs, _ := filepath.Abs(path)
			rel, _ := filepath.Rel(root, path)

			// Skip hidden files and directories.
			if f.IsDir() || rel == "." || strings.HasPrefix(rel, ".private") || strings.HasPrefix(rel, "admin/") {
				return nil
			}

			// Hide vendored files.
			if i == 1 && strings.HasPrefix(rel, "js/ace-editor") {
				return nil
			}

			// Only text files.
			ext := strings.ToLower(filepath.Ext(path))
			okTypes := []string{
				".html", ".gohtml", ".md", ".markdown", ".js", ".css", ".jsx",
			}
			ok := false
			for _, ft := range okTypes {
				if ext == ft {
					ok = true
					break
				}
			}
			if !ok {
				return nil
			}

			tree.Files = append(tree.Files, render.Filepath{
				Absolute: abs,
				Relative: rel,
				Basename: filepath.Base(path),
			})
			return nil
		})

		trees = append(trees, tree)
	}
	v := map[string]interface{}{
		"FileTrees": trees,
	}
	render.Template(w, r, "admin/filelist", v)
}
