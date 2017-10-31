package core

import (
	"html/template"
	"net/http"
)

// DefaultVars combines template variables with default, globally available vars.
func (b *Blog) DefaultVars(vars map[string]interface{}) map[string]interface{} {
	defaults := map[string]interface{}{
		"title": "Untitled Blog",
	}
	if vars == nil {
		return defaults
	}

	for k, v := range defaults {
		vars[k] = v
	}

	return vars
}

// RenderTemplate responds with an HTML template.
func (b *Blog) RenderTemplate(w http.ResponseWriter, r *http.Request, path string, vars map[string]interface{}) error {
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
	vars = b.DefaultVars(vars)

	w.Header().Set("Content-Type", "text/html; encoding=UTF-8")
	err = t.ExecuteTemplate(w, "layout", vars)
	if err != nil {
		log.Error("Template parsing error: %s", err)
		return err
	}

	log.Debug("Parsed template")

	return nil
}
