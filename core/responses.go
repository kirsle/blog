package core

import (
	"fmt"
	"net/http"

	"github.com/kirsle/blog/core/internal/log"
	"github.com/kirsle/blog/core/internal/render"
	"github.com/kirsle/blog/core/internal/sessions"
)

// Flash adds a flash message to the user's session.
func (b *Blog) Flash(w http.ResponseWriter, r *http.Request, message string, args ...interface{}) {
	session := sessions.Get(r)
	session.AddFlash(fmt.Sprintf(message, args...))
	session.Save(r, w)
}

// FlashAndRedirect flashes and redirects in one go.
func (b *Blog) FlashAndRedirect(w http.ResponseWriter, r *http.Request, location, message string, args ...interface{}) {
	b.Flash(w, r, message, args...)
	b.Redirect(w, location)
}

// FlashAndReload flashes and sends a redirect to the same path.
func (b *Blog) FlashAndReload(w http.ResponseWriter, r *http.Request, message string, args ...interface{}) {
	b.Flash(w, r, message, args...)
	b.Redirect(w, r.URL.Path)
}

// Redirect sends an HTTP redirect response.
func (b *Blog) Redirect(w http.ResponseWriter, location string) {
	w.Header().Set("Location", location)
	w.WriteHeader(http.StatusFound)
}

// NotFound sends a 404 response.
func (b *Blog) NotFound(w http.ResponseWriter, r *http.Request, message string) {
	if message == "" {
		message = "The page you were looking for was not found."
	}

	w.WriteHeader(http.StatusNotFound)
	err := b.RenderTemplate(w, r, ".errors/404", render.Vars{
		Message: message,
	})
	if err != nil {
		log.Error(err.Error())
		w.Write([]byte("Unrecoverable template error for NotFound()"))
	}
}

// Forbidden sends an HTTP 403 Forbidden response.
func (b *Blog) Forbidden(w http.ResponseWriter, r *http.Request, message string) {
	w.WriteHeader(http.StatusForbidden)
	err := b.RenderTemplate(w, r, ".errors/403", render.Vars{
		Message: message,
	})
	if err != nil {
		log.Error(err.Error())
		w.Write([]byte("Unrecoverable template error for Forbidden()"))
	}
}

// Error sends an HTTP 500 Internal Server Error response.
func (b *Blog) Error(w http.ResponseWriter, r *http.Request, message string) {
	w.WriteHeader(http.StatusInternalServerError)
	err := b.RenderTemplate(w, r, ".errors/500", render.Vars{
		Message: message,
	})
	if err != nil {
		log.Error(err.Error())
		w.Write([]byte("Unrecoverable template error for Error()"))
	}
}

// BadRequest sends an HTTP 400 Bad Request.
func (b *Blog) BadRequest(w http.ResponseWriter, r *http.Request, message string) {
	w.WriteHeader(http.StatusBadRequest)
	err := b.RenderTemplate(w, r, ".errors/400", render.Vars{
		Message: message,
	})
	if err != nil {
		log.Error(err.Error())
		w.Write([]byte("Unrecoverable template error for BadRequest()"))
	}
}
