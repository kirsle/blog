package render

import (
	"html/template"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/kirsle/blog/core/internal/forms"
	"github.com/kirsle/blog/core/internal/log"
	"github.com/kirsle/blog/core/internal/middleware"
	"github.com/kirsle/blog/core/internal/middleware/auth"
	"github.com/kirsle/blog/core/internal/models/settings"
	"github.com/kirsle/blog/core/internal/models/users"
	"github.com/kirsle/blog/core/internal/sessions"
	"github.com/kirsle/blog/core/internal/types"
)

// Vars is an interface to implement by the templates to pass their own custom
// variables in. It auto-loads global template variables (site name, etc.)
// when the template is rendered.
type Vars struct {
	// Global, "constant" template variables.
	SetupNeeded     bool
	Title           string
	Path            string
	TemplatePath    string
	LoggedIn        bool
	CurrentUser     *users.User
	CSRF            string
	Editable        bool // page is editable
	Request         *http.Request
	RequestTime     time.Time
	RequestDuration time.Duration

	// Configuration variables
	NoLayout bool // don't wrap in .layout.html, just render the template

	// Common template variables.
	Message string
	Flashes []string
	Error   error
	Data    map[interface{}]interface{}
	Form    forms.Form
}

// loadDefaults combines template variables with default, globally available vars.
func (v *Vars) loadDefaults(r *http.Request) {
	// Get the site settings.
	s, err := settings.Load()
	if err != nil {
		s = settings.Defaults()
	}

	if s.Initialized == false && !strings.HasPrefix(r.URL.Path, "/initial-setup") {
		v.SetupNeeded = true
	}
	v.Request = r
	v.RequestTime = r.Context().Value(types.StartTimeKey).(time.Time)
	v.Title = s.Site.Title
	v.Path = r.URL.Path

	user, err := auth.CurrentUser(r)
	v.CurrentUser = user
	v.LoggedIn = err == nil
}

// Template responds with an HTML template.
//
// The vars will be massaged a bit to load the global defaults (such as the
// website title and user login status), the user's session may be updated with
// new CSRF token, and other such things. If you just want to render a template
// without all that nonsense, use RenderPartialTemplate.
func Template(w io.Writer, r *http.Request, path string, v Vars) error {
	// Inject globally available variables.
	v.loadDefaults(r)

	// If this is the HTTP response, handle session-related things.
	if rw, ok := w.(http.ResponseWriter); ok {
		rw.Header().Set("Content-Type", "text/html; encoding=UTF-8")
		session := sessions.Get(r)

		// Flashed messages.
		if flashes := session.Flashes(); len(flashes) > 0 {
			for _, flash := range flashes {
				_ = flash
				v.Flashes = append(v.Flashes, flash.(string))
			}
			session.Save(r, rw)
		}

		// CSRF token for forms.
		v.CSRF = middleware.GenerateCSRFToken(rw, r, session)
	}

	v.RequestDuration = time.Now().Sub(v.RequestTime)
	v.Editable = !strings.HasPrefix(path, "admin/")

	// v interface{}, withLayout bool, functions map[string]interface{}) error {
	var (
		layout       Filepath
		templateName string
		err          error
	)

	// Find the file path to the template.
	filepath, err := ResolvePath(path)
	if err != nil {
		log.Error("RenderTemplate(%s): file not found", path)
		return err
	}

	// Get the layout template.
	if !v.NoLayout {
		templateName = "layout"
		layout, err = ResolvePath(".layout")
		if err != nil {
			log.Error("RenderTemplate(%s): layout template not found", path)
			return err
		}
	} else {
		templateName = filepath.Basename
	}

	// The comment entry partial.
	commentEntry, err := ResolvePath("comments/entry.partial")
	if err != nil {
		log.Error("RenderTemplate(%s): comments/entry.partial not found")
		return err
	}

	t := template.New(filepath.Absolute).Funcs(Funcs)

	// Parse the template files. The layout comes first because it's the wrapper
	// and allows the filepath template to set the page title.
	var templates []string
	if !v.NoLayout {
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
