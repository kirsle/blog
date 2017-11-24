package core

import (
	"html/template"
	"net/http"
	"strings"

	"github.com/kirsle/blog/core/forms"
	"github.com/kirsle/blog/core/models/settings"
	"github.com/kirsle/blog/core/models/users"
)

// Vars is an interface to implement by the templates to pass their own custom
// variables in. It auto-loads global template variables (site name, etc.)
// when the template is rendered.
type Vars struct {
	// Global template variables.
	SetupNeeded bool
	Title       string
	Path        string
	LoggedIn    bool
	CurrentUser *users.User
	CSRF        string
	Request     *http.Request

	// Common template variables.
	Message string
	Flashes []string
	Error   error
	Data    map[interface{}]interface{}
	Form    forms.Form
}

// NewVars initializes a Vars struct with the custom Data map initialized.
// You may pass in an initial value for this map if you want.
func NewVars(data ...map[interface{}]interface{}) *Vars {
	var value map[interface{}]interface{}
	if len(data) > 0 {
		value = data[0]
	} else {
		value = make(map[interface{}]interface{})
	}
	return &Vars{
		Data: value,
	}
}

// LoadDefaults combines template variables with default, globally available vars.
func (v *Vars) LoadDefaults(b *Blog, w http.ResponseWriter, r *http.Request) {
	// Get the site settings.
	s, err := settings.Load()
	if err != nil {
		s = settings.Defaults()
	}

	if s.Initialized == false && !strings.HasPrefix(r.URL.Path, "/initial-setup") {
		v.SetupNeeded = true
	}
	v.Request = r
	v.Title = s.Site.Title
	v.Path = r.URL.Path

	user, err := b.CurrentUser(r)
	v.CurrentUser = user
	v.LoggedIn = err == nil

	// Add any flashed messages from the endpoint controllers.
	session := b.Session(r)
	if flashes := session.Flashes(); len(flashes) > 0 {
		for _, flash := range flashes {
			_ = flash
			v.Flashes = append(v.Flashes, flash.(string))
		}
		session.Save(r, w)
	}

	v.CSRF = b.GenerateCSRFToken(w, r, session)
}

// TemplateVars is an interface that describes the template variable struct.
type TemplateVars interface {
	LoadDefaults(*Blog, http.ResponseWriter, *http.Request)
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

	// Useful template functions.
	log.Error("HERE!!!")
	t := template.New(filepath.Absolute).Funcs(template.FuncMap{
		"StringsJoin": strings.Join,
		"RenderPost":  b.RenderPost,
	})

	// Parse the template files. The layout comes first because it's the wrapper
	// and allows the filepath template to set the page title.
	t, err = t.ParseFiles(layout.Absolute, filepath.Absolute)
	if err != nil {
		log.Error(err.Error())
		return err
	}

	// Inject globally available variables.
	if vars == nil {
		vars = &Vars{}
	}
	vars.LoadDefaults(b, w, r)

	w.Header().Set("Content-Type", "text/html; encoding=UTF-8")
	err = t.ExecuteTemplate(w, "layout", vars)
	if err != nil {
		log.Error("Template parsing error: %s", err)
		return err
	}

	log.Debug("Parsed template")

	return nil
}
