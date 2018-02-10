package render

import (
	"html/template"
	"strings"
	"time"
)

// Funcs is a global funcmap that the blog can hook its internal
// methods onto.
var Funcs = template.FuncMap{
	"StringsJoin": strings.Join,
	"Now":         time.Now,
}
