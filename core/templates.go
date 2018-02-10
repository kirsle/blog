package core

import (
	"io"
	"net/http"

	"github.com/kirsle/blog/core/internal/render"
)

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

// RenderTemplate responds with an HTML template.
//
// The vars will be massaged a bit to load the global defaults (such as the
// website title and user login status), the user's session may be updated with
// new CSRF token, and other such things.
//
// For server-rendered templates given directly to the user (i.e., in controllers),
// give it the http.ResponseWriter; for partial templates you can give it a
// bytes.Buffer to write to instead. The subtle difference is whether or not the
// template will have access to the request's session.
func (b *Blog) RenderTemplate(w io.Writer, r *http.Request, path string, vars render.Vars) error {
	if r == nil {
		panic("core.RenderTemplate(): the *http.Request is nil!?")
	}

	return render.Template(w, r, path, vars)
}
