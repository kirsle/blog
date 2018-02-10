package render

import (
	"html/template"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/kirsle/blog/core/internal/forms"
	"github.com/kirsle/blog/core/internal/log"
	"github.com/kirsle/blog/core/internal/models/users"
)

// Config provides the settings and injectables for rendering templates.
type Config struct {
	// Refined and raw variables for the templates.
	Vars *Vars // Normal RenderTemplate's

	// Wrap the template with the `.layout.gohtml`
	WithLayout bool

	// Inject your own functions for the Go templates.
	Functions map[string]interface{}

	Request *http.Request
}

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

// PartialTemplate handles rendering a Go template to a writer, without
// doing anything extra to the vars or dealing with net/http. This is ideal for
// rendering partials, such as comment partials.
//
// This will wrap the template in `.layout.gohtml` by default. To render just
// a bare template on its own, i.e. for partial templates, create a Vars struct
// with `Vars{NoIndex: true}`
func PartialTemplate(w io.Writer, path string, C Config) error {
	if C.Request == nil {
		panic("render.RenderPartialTemplate(): The *http.Request is nil!?")
	}

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
	if C.WithLayout {
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

	// Template functions.
	funcmap := template.FuncMap{
		"StringsJoin": strings.Join,
		"Now":         time.Now,
		"TemplateName": func() string {
			return filepath.URI
		},
	}
	if C.Functions != nil {
		for name, fn := range C.Functions {
			funcmap[name] = fn
		}
	}

	// Useful template functions.
	t := template.New(filepath.Absolute).Funcs(funcmap)

	// Parse the template files. The layout comes first because it's the wrapper
	// and allows the filepath template to set the page title.
	var templates []string
	if C.WithLayout {
		templates = append(templates, layout.Absolute)
	}
	t, err = t.ParseFiles(append(templates, commentEntry.Absolute, filepath.Absolute)...)
	if err != nil {
		log.Error(err.Error())
		return err
	}

	err = t.ExecuteTemplate(w, templateName, C.Vars)
	if err != nil {
		log.Error("Template parsing error: %s", err)
		return err
	}

	return nil
}

// Template responds with an HTML template.
//
// The vars will be massaged a bit to load the global defaults (such as the
// website title and user login status), the user's session may be updated with
// new CSRF token, and other such things. If you just want to render a template
// without all that nonsense, use RenderPartialTemplate.
func Template(w http.ResponseWriter, path string, C Config) error {
	if C.Request == nil {
		panic("render.RenderTemplate(): The *http.Request is nil!?")
	}

	w.Header().Set("Content-Type", "text/html; encoding=UTF-8")
	PartialTemplate(w, path, Config{
		Request:    C.Request,
		Vars:       C.Vars,
		WithLayout: true,
		Functions:  C.Functions,
	})

	return nil
}
