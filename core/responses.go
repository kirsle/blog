package core

import (
	"net/http"
)

// Redirect sends an HTTP redirect response.
func (b *Blog) Redirect(w http.ResponseWriter, location string) {
	w.Header().Set("Location", location)
	w.WriteHeader(http.StatusFound)
}

// NotFound sends a 404 response.
func (b *Blog) NotFound(w http.ResponseWriter, r *http.Request, message ...string) {
	if len(message) == 0 {
		message = []string{"The page you were looking for was not found."}
	}

	w.WriteHeader(http.StatusNotFound)
	err := b.RenderTemplate(w, r, ".errors/404", map[string]interface{}{
		"message": message[0],
	})
	if err != nil {
		log.Error(err.Error())
		http.NotFound(w, r)
	}
}

// Forbidden sends an HTTP 400 Forbidden response.
func (b *Blog) Forbidden(w http.ResponseWriter, r *http.Request, message ...string) {
	w.WriteHeader(http.StatusForbidden)
	err := b.RenderTemplate(w, r, ".errors/403", nil)
	if err != nil {
		http.NotFound(w, r)
	}
}
