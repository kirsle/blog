package core

import (
	"html/template"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/kirsle/blog/core/internal/markdown"
	"github.com/kirsle/blog/core/internal/render"
	"github.com/kirsle/blog/core/internal/responses"
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
		b.Forbidden(w, r, "Forbidden")
		return
	}

	// Search for a file that matches their URL.
	filepath, err := render.ResolvePath(path)
	if err != nil {
		// See if it resolves as a blog entry.
		err = b.viewPost(w, r, strings.TrimLeft(path, "/"))
		if err != nil {
			b.NotFound(w, r, "The page you were looking for was not found.")
		}
		return
	}

	// Is it a template file?
	if strings.HasSuffix(filepath.URI, ".gohtml") {
		b.RenderTemplate(w, r, filepath.URI, NewVars())
		return
	}

	// Is it a Markdown file?
	if strings.HasSuffix(filepath.URI, ".md") || strings.HasSuffix(filepath.URI, ".markdown") {
		source, err := ioutil.ReadFile(filepath.Absolute)
		if err != nil {
			b.Error(w, r, "Couldn't read Markdown source!")
			return
		}

		// Render it to HTML and find out its title.
		body := string(source)
		html := markdown.RenderTrustedMarkdown(body)
		title, _ := markdown.TitleFromMarkdown(body)

		b.RenderTemplate(w, r, ".markdown", NewVars(map[interface{}]interface{}{
			"Title":        title,
			"HTML":         template.HTML(html),
			"MarkdownFile": filepath.URI,
		}))
		return
	}

	http.ServeFile(w, r, filepath.Absolute)
}
