package blog

import (
	"html/template"
	"io/ioutil"
	"mime"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/kirsle/blog/src/controllers/posts"
	"github.com/kirsle/blog/src/log"
	"github.com/kirsle/blog/src/markdown"
	"github.com/kirsle/blog/src/render"
	"github.com/kirsle/blog/src/responses"
	"github.com/kirsle/blog/src/root"
)

// PageHandler is the catch-all route handler, for serving static web pages.
func (b *Blog) PageHandler(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path
	// log.Debug("Catch-all page handler invoked for request URI: %s", path)

	// Remove trailing slashes by redirecting them away.
	if len(path) > 1 && path[len(path)-1] == '/' {
		responses.Redirect(w, strings.TrimRight(path, "/"))
		return
	}

	// Restrict special paths.
	if strings.HasPrefix(strings.ToLower(path), "/.") {
		responses.Forbidden(w, r, "Forbidden")
		return
	}

	// Search for a file that matches their URL.
	fp, err := render.ResolvePath(path)
	if err != nil {
		// See if it resolves as a blog entry.
		err = postctl.ViewPost(w, r, strings.TrimLeft(path, "/"))
		if err != nil {
			log.Error("Post by fragment %s not found: %s", path, err)
			responses.NotFound(w, r, "The page you were looking for was not found.")
		}
		return
	}

	// Is it a template file?
	if strings.HasSuffix(fp.URI, ".gohtml") {
		render.Template(w, r, fp.URI, nil)
		return
	}

	// Is it a Markdown file?
	if strings.HasSuffix(fp.URI, ".md") || strings.HasSuffix(fp.URI, ".markdown") {
		var source []byte
		if len(fp.BindataKey) > 0 {
			data, err := root.Asset(fp.BindataKey)
			if err != nil {
				responses.Error(w, r, "Couldn't read bindata key: "+fp.BindataKey)
				return
			}

			source = data
		} else {
			data, err := ioutil.ReadFile(fp.Absolute)
			if err != nil {
				responses.Error(w, r, "Couldn't read Markdown source!")
				return
			}

			source = data
		}

		// Render it to HTML and find out its title.
		body := string(source)
		html := markdown.RenderTrustedMarkdown(body)
		title, _ := markdown.TitleFromMarkdown(body)

		render.Template(w, r, ".markdown", map[string]interface{}{
			"Title":        title,
			"HTML":         template.HTML(html),
			"MarkdownPath": fp.URI,
		})
		return
	}

	// It's a regular static file we can serve directly.
	{
		// Check if we have bindata for it.
		if fp.BindataKey != "" {
			data, _ := root.Asset(fp.BindataKey)
			w.Header().Set("Content-Type", mime.TypeByExtension(filepath.Ext(fp.URI)))
			w.Write(data)
			return
		}

		// Try the filesystem.
		http.ServeFile(w, r, fp.Absolute)
	}
}
