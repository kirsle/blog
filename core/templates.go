package core

import (
	"html/template"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/kirsle/blog/core/forms"
	"github.com/kirsle/blog/core/models/settings"
	"github.com/kirsle/blog/core/models/users"
)

// Vars is an interface to implement by the templates to pass their own custom
// variables in. It auto-loads global template variables (site name, etc.)
// when the template is rendered.
type Vars struct {
	// Global, "constant" template variables.
	SetupNeeded bool
	Title       string
	Path        string
	LoggedIn    bool
	CurrentUser *users.User
	CSRF        string
	Request     *http.Request

	// Configuration variables
	NoLayout bool // don't wrap in .layout.html, just render the template

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
func (v *Vars) LoadDefaults(b *Blog, r *http.Request) {
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
}

// // TemplateVars is an interface that describes the template variable struct.
// type TemplateVars interface {
// 	LoadDefaults(*Blog, *http.Request)
// }

// RenderPartialTemplate handles rendering a Go template to a writer, without
// doing anything extra to the vars or dealing with net/http. This is ideal for
// rendering partials, such as comment partials.
//
// This will wrap the template in `.layout.gohtml` by default. To render just
// a bare template on its own, i.e. for partial templates, create a Vars struct
// with `Vars{NoIndex: true}`
func (b *Blog) RenderPartialTemplate(w io.Writer, path string, v interface{}, withLayout bool, functions map[string]interface{}) error {
	var (
		layout       Filepath
		templateName string
		err          error
	)

	// Find the file path to the template.
	filepath, err := b.ResolvePath(path)
	if err != nil {
		log.Error("RenderTemplate(%s): file not found", path)
		return err
	}

	// Get the layout template.
	if withLayout {
		templateName = "layout"
		layout, err = b.ResolvePath(".layout")
		if err != nil {
			log.Error("RenderTemplate(%s): layout template not found", path)
			return err
		}
	} else {
		templateName = filepath.Basename
	}

	// The comment entry partial.
	commentEntry, err := b.ResolvePath("comments/entry.partial")
	if err != nil {
		log.Error("RenderTemplate(%s): comments/entry.partial not found")
		return err
	}

	// Template functions.
	funcmap := template.FuncMap{
		"StringsJoin": strings.Join,
		"Now":         time.Now,
		"RenderIndex": b.RenderIndex,
		"RenderPost":  b.RenderPost,
	}
	if functions != nil {
		for name, fn := range functions {
			funcmap[name] = fn
		}
	}

	// Useful template functions.
	t := template.New(filepath.Absolute).Funcs(funcmap)

	// Parse the template files. The layout comes first because it's the wrapper
	// and allows the filepath template to set the page title.
	var templates []string
	if withLayout {
		templates = append(templates, layout.Absolute)
	}
	t, err = t.ParseFiles(append(templates, commentEntry.Absolute, filepath.Absolute)...)
	if err != nil {
		log.Error(err.Error())
		return err
	}

	err = t.ExecuteTemplate(w, templateName, v)
	if err != nil {
		log.Error("Template parsing error: %s", err)
		return err
	}

	return nil
}

// RenderTemplate responds with an HTML template.
//
// The vars will be massaged a bit to load the global defaults (such as the
// website title and user login status), the user's session may be updated with
// new CSRF token, and other such things. If you just want to render a template
// without all that nonsense, use RenderPartialTemplate.
func (b *Blog) RenderTemplate(w http.ResponseWriter, r *http.Request, path string, vars *Vars) error {
	// Inject globally available variables.
	if vars == nil {
		vars = &Vars{}
	}
	vars.LoadDefaults(b, r)

	// Add any flashed messages from the endpoint controllers.
	session := b.Session(r)
	if flashes := session.Flashes(); len(flashes) > 0 {
		for _, flash := range flashes {
			_ = flash
			vars.Flashes = append(vars.Flashes, flash.(string))
		}
		session.Save(r, w)
	}

	vars.CSRF = b.GenerateCSRFToken(w, r, session)

	w.Header().Set("Content-Type", "text/html; encoding=UTF-8")
	b.RenderPartialTemplate(w, path, vars, true, template.FuncMap{
		"RenderComments": func(subject string, ids ...string) template.HTML {
			session := b.Session(r)
			csrf := b.GenerateCSRFToken(w, r, session)
			return b.RenderComments(session, csrf, r.URL.Path, subject, ids...)
		},
	})
	log.Debug("Parsed template")

	return nil
}
