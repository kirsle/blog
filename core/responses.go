package core

import (
	"net/http"
)

// Redirect sends an HTTP redirect response.
func (b *Blog) Redirect(w http.ResponseWriter, location string) {
	log.Error("Redirect: %s", location)
	w.Header().Set("Location", location)
	w.WriteHeader(http.StatusFound)
}

// NotFound sends a 404 response.
func (b *Blog) NotFound(w http.ResponseWriter, r *http.Request, message ...string) {
	if len(message) == 0 {
		message = []string{"The page you were looking for was not found."}
	}

	log.Error("HERE 2")
	w.WriteHeader(http.StatusNotFound)
	err := b.RenderTemplate(w, r, ".errors/404", &Vars{
		Message: message[0],
	})
	if err != nil {
		log.Error(err.Error())
		w.Write([]byte("Unrecoverable template error for NotFound()"))
	}
}

// Forbidden sends an HTTP 403 Forbidden response.
func (b *Blog) Forbidden(w http.ResponseWriter, r *http.Request, message ...string) {
	log.Error("HERE 3")
	w.WriteHeader(http.StatusForbidden)
	err := b.RenderTemplate(w, r, ".errors/403", nil)
	if err != nil {
		log.Error(err.Error())
		w.Write([]byte("Unrecoverable template error for Forbidden()"))
	}
}

// BadRequest sends an HTTP 400 Bad Request.
func (b *Blog) BadRequest(w http.ResponseWriter, r *http.Request, message ...string) {
	log.Error("HERE 4")
	w.WriteHeader(http.StatusBadRequest)
	err := b.RenderTemplate(w, r, ".errors/400", &Vars{
		Message: message[0],
	})
	if err != nil {
		log.Error(err.Error())
		w.Write([]byte("Unrecoverable template error for BadRequest()"))
	}
}
