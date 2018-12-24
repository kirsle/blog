package postctl

import (
	"bytes"
	"html/template"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/kirsle/blog/models/posts"
	"github.com/kirsle/blog/src/render"
)

// tagged lets you browse blog posts by category.
func taggedHandler(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	tag, ok := params["tag"]
	if !ok {
		// They're listing all the tags.
		render.Template(w, r, "blog/tags.gohtml", nil)
		return
	}

	commonIndexHandler(w, r, tag, "")
}

// partialTags renders the tags partial.
func partialTags(r *http.Request, indexView bool) template.HTML {
	idx, err := posts.GetIndex()
	if err != nil {
		return template.HTML("[RenderTags: error getting blog index]")
	}

	tags, err := idx.Tags()
	if err != nil {
		return template.HTML("[RenderTags: error getting tags]")
	}

	var output bytes.Buffer
	v := map[string]interface{}{
		"IndexView": indexView,
		"Tags":      tags,
	}
	render.Template(&output, r, "blog/tags.partial", v)

	return template.HTML(output.String())
}
