package core

import (
	"html/template"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/kirsle/blog/core/internal/forms"
	"github.com/kirsle/blog/core/internal/middleware"
	"github.com/kirsle/blog/core/internal/middleware/auth"
	"github.com/kirsle/blog/core/internal/models/settings"
	"github.com/kirsle/blog/core/internal/models/users"
	"github.com/kirsle/blog/core/internal/render"
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

// NewVars initializes a Vars struct with the custom Data map initialized.
// You may pass in an initial value for this map if you want.
func NewVars(data ...map[interface{}]interface{}) render.Vars {
	var value map[interface{}]interface{}
	if len(data) > 0 {
		value = data[0]
	} else {
		value = make(map[interface{}]interface{})
	}
	return render.Vars{
		Data: value,
	}
}

// LoadDefaults combines template variables with default, globally available vars.
func (b *Blog) LoadDefaults(v render.Vars, r *http.Request) render.Vars {
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

	return v
}

// RenderPartialTemplate handles rendering a Go template to a writer, without
// doing anything extra to the vars or dealing with net/http. This is ideal for
// rendering partials, such as comment partials.
//
// This will wrap the template in `.layout.gohtml` by default. To render just
// a bare template on its own, i.e. for partial templates, create a Vars struct
// with `Vars{NoIndex: true}`
func (b *Blog) RenderPartialTemplate(w io.Writer, r *http.Request, path string, v render.Vars, withLayout bool, functions map[string]interface{}) error {
	v = b.LoadDefaults(v, r)
	return render.PartialTemplate(w, path, render.Config{
		Request:    r,
		Vars:       &v,
		WithLayout: withLayout,
		Functions:  b.TemplateFuncs(nil, nil, functions),
	})
}

// RenderTemplate responds with an HTML template.
//
// The vars will be massaged a bit to load the global defaults (such as the
// website title and user login status), the user's session may be updated with
// new CSRF token, and other such things. If you just want to render a template
// without all that nonsense, use RenderPartialTemplate.
func (b *Blog) RenderTemplate(w http.ResponseWriter, r *http.Request, path string, vars render.Vars) error {
	if r == nil {
		panic("core.RenderTemplate(): the *http.Request is nil!?")
	}

	// Inject globally available variables.
	vars = b.LoadDefaults(vars, r)

	// Add any flashed messages from the endpoint controllers.
	session := sessions.Get(r)
	if flashes := session.Flashes(); len(flashes) > 0 {
		for _, flash := range flashes {
			_ = flash
			vars.Flashes = append(vars.Flashes, flash.(string))
		}
		session.Save(r, w)
	}

	vars.RequestDuration = time.Now().Sub(vars.RequestTime)
	vars.CSRF = middleware.GenerateCSRFToken(w, r, session)
	vars.Editable = !strings.HasPrefix(path, "admin/")

	return render.Template(w, path, render.Config{
		Request:   r,
		Vars:      &vars,
		Functions: b.TemplateFuncs(w, r, nil),
	})
}

// TemplateFuncs returns the common template function map.
func (b *Blog) TemplateFuncs(w http.ResponseWriter, r *http.Request, inject map[string]interface{}) map[string]interface{} {
	fn := map[string]interface{}{
		"RenderIndex": b.RenderIndex,
		"RenderPost":  b.RenderPost,
		"RenderTags":  b.RenderTags,
		"RenderComments": func(subject string, ids ...string) template.HTML {
			if w == nil || r == nil {
				return template.HTML("[RenderComments Error: need both http.ResponseWriter and http.Request]")
			}

			session := sessions.Get(r)
			csrf := middleware.GenerateCSRFToken(w, r, session)
			return b.RenderComments(session, csrf, r.URL.Path, subject, ids...)
		},
	}

	if inject != nil {
		for k, v := range inject {
			fn[k] = v
		}
	}
	return fn
}
