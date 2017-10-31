package core

import (
	"net/http"
)

// SetupHandler is the initial blog setup route.
func (b *Blog) SetupHandler(w http.ResponseWriter, r *http.Request) {
	b.RenderTemplate(w, r, "admin/setup", nil)
}
