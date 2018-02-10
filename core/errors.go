package core

import (
	"net/http"

	"github.com/kirsle/blog/core/internal/log"
	"github.com/kirsle/blog/core/internal/render"
)

// NotFound sends a 404 response.
func (b *Blog) NotFound(w http.ResponseWriter, r *http.Request, message string) {
	if message == "" {
		message = "The page you were looking for was not found."
	}

	w.WriteHeader(http.StatusNotFound)
	err := render.Template(w, r, ".errors/404", map[string]string{
		"Message": message,
	})
	if err != nil {
		log.Error(err.Error())
		w.Write([]byte("Unrecoverable template error for NotFound()"))
	}
}

// Forbidden sends an HTTP 403 Forbidden response.
func (b *Blog) Forbidden(w http.ResponseWriter, r *http.Request, message string) {
	w.WriteHeader(http.StatusForbidden)
	err := render.Template(w, r, ".errors/403", map[string]string{
		"Message": message,
	})
	if err != nil {
		log.Error(err.Error())
		w.Write([]byte("Unrecoverable template error for Forbidden()"))
	}
}

// Error sends an HTTP 500 Internal Server Error response.
func (b *Blog) Error(w http.ResponseWriter, r *http.Request, message string) {
	w.WriteHeader(http.StatusInternalServerError)
	err := render.Template(w, r, ".errors/500", map[string]string{
		"Message": message,
	})
	if err != nil {
		log.Error(err.Error())
		w.Write([]byte("Unrecoverable template error for Error()"))
	}
}

// BadRequest sends an HTTP 400 Bad Request.
func (b *Blog) BadRequest(w http.ResponseWriter, r *http.Request, message string) {
	w.WriteHeader(http.StatusBadRequest)
	err := render.Template(w, r, ".errors/400", map[string]string{
		"Message": message,
	})
	if err != nil {
		log.Error(err.Error())
		w.Write([]byte("Unrecoverable template error for BadRequest()"))
	}
}
