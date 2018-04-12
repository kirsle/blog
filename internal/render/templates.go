package render

import (
	"html/template"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/kirsle/blog/internal/log"
	"github.com/kirsle/blog/internal/middleware"
	"github.com/kirsle/blog/internal/middleware/auth"
	"github.com/kirsle/blog/internal/sessions"
	"github.com/kirsle/blog/internal/types"
	"github.com/kirsle/blog/models/settings"
	"github.com/kirsle/blog/models/users"
)

// Vars is an interface to implement by the templates to pass their own custom
// variables in. It auto-loads global template variables (site name, etc.)
// when the template is rendered.
type vars struct {
	// Global, "constant" template variables.
	SetupNeeded     bool
	Title           string
	Description     string
	Path            string
	TemplatePath    string // actual template file on disk
	LoggedIn        bool
	CurrentUser     *users.User
	CSRF            string
	Editable        bool // page is editable
	Request         *http.Request
	RequestTime     time.Time
	RequestDuration time.Duration

	// Common template variables.
	Message string
	Flashes []string
	Error   error
	Data    interface{}
}

// Template responds with an HTML template.
//
// The vars will be massaged a bit to load the global defaults (such as the
// website title and user login status), the user's session may be updated with
// new CSRF token, and other such things. If you just want to render a template
// without all that nonsense, use RenderPartialTemplate.
func Template(w io.Writer, r *http.Request, path string, data interface{}) error {
	isPartial := strings.Contains(path, ".partial")

	// Get the site settings.
	s, err := settings.Load()
	if err != nil {
		s = settings.Defaults()
	}

	// Inject globally available variables.
	v := vars{
		SetupNeeded: s.Initialized == false && !strings.HasPrefix(r.URL.Path, "/initial-setup"),

		Request:     r,
		RequestTime: r.Context().Value(types.StartTimeKey).(time.Time),
		Title:       s.Site.Title,
		Description: s.Site.Description,
		Path:        r.URL.Path,

		Data: data,
	}

	user, err := auth.CurrentUser(r)
	v.CurrentUser = user
	v.LoggedIn = err == nil

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

	var (
		layout       Filepath
		templateName string
	)

	// Find the file path to the template.
	filepath, err := ResolvePath(path)
	if err != nil {
		log.Error("RenderTemplate(%s): file not found", path)
		return err
	}
	v.TemplatePath = filepath.URI

	// Get the layout template.
	if !isPartial {
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
	if !isPartial {
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
