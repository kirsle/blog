package core

import (
	"html/template"
	"net/http"

	"github.com/kirsle/blog/core/forms"
	"github.com/kirsle/blog/core/models/users"
)

// Vars is an interface to implement by the templates to pass their own custom
// variables in. It auto-loads global template variables (site name, etc.)
// when the template is rendered.
type Vars struct {
	// Global template variables.
	Title       string
	LoggedIn    bool
	CurrentUser *users.User

	// Common template variables.
	Message string
	Flash   string
	Error   error
	Form    forms.Form
}

// LoadDefaults combines template variables with default, globally available vars.
func (v *Vars) LoadDefaults(r *http.Request) {
	v.Title = "Untitled Blog"

	ctx := r.Context()
	if user, ok := ctx.Value(userKey).(*users.User); ok {
		if user.ID > 0 {
			v.LoggedIn = true
			v.CurrentUser = user
		}
	}
}

// TemplateVars is an interface that describes the template variable struct.
type TemplateVars interface {
	LoadDefaults(*http.Request)
}

// RenderTemplate responds with an HTML template.
func (b *Blog) RenderTemplate(w http.ResponseWriter, r *http.Request, path string, vars TemplateVars) error {
	// Get the layout template.
	layout, err := b.ResolvePath(".layout")
	if err != nil {
		log.Error("RenderTemplate(%s): layout template not found", path)
		return err
	}

	// And the template in question.
	filepath, err := b.ResolvePath(path)
	if err != nil {
		log.Error("RenderTemplate(%s): file not found", path)
		return err
	}

	// Parse the template files. The layout comes first because it's the wrapper
	// and allows the filepath template to set the page title.
	t, err := template.ParseFiles(layout.Absolute, filepath.Absolute)
	if err != nil {
		log.Error(err.Error())
		return err
	}

	// Inject globally available variables.
	if vars == nil {
		vars = &Vars{}
	}
	vars.LoadDefaults(r)

	w.Header().Set("Content-Type", "text/html; encoding=UTF-8")
	err = t.ExecuteTemplate(w, "layout", vars)
	if err != nil {
		log.Error("Template parsing error: %s", err)
		return err
	}

	log.Debug("Parsed template")

	return nil
}
