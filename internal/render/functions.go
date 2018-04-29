package render

import (
	"html/template"
	"strings"
	"time"

	"github.com/kirsle/blog/internal/markdown"
)

// Funcs is a global funcmap that the blog can hook its internal
// methods onto.
var Funcs = template.FuncMap{
	"StringsJoin": strings.Join,
	"NewlinesToSpace": func(text string) string {
		return strings.Replace(
			strings.Replace(text, "\n", " ", -1),
			"\r", "", -1,
		)
	},
	"Now": time.Now,
	"TrustedMarkdown": func(text string) template.HTML {
		return template.HTML(markdown.RenderTrustedMarkdown(text))
	},
}
