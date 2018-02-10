package responses

import (
	"net/http"

	"github.com/kirsle/blog/core/internal/log"
	"github.com/kirsle/blog/core/internal/render"
)

// Redirect sends an HTTP redirect response.
func Redirect(w http.ResponseWriter, location string) {
	w.Header().Set("Location", location)
	w.WriteHeader(http.StatusFound)
}

// NotFound sends a 404 response.
func NotFound(w http.ResponseWriter, r *http.Request, message ...string) {
	if len(message) == 0 {
		message = []string{"The page you were looking for was not found."}
	}

	w.WriteHeader(http.StatusNotFound)
	err := render.RenderTemplate(w, r, ".errors/404", &render.Vars{
		Message: message[0],
	})
	if err != nil {
		log.Error(err.Error())
		w.Write([]byte("Unrecoverable template error for NotFound()"))
	}
}

// Forbidden sends an HTTP 403 Forbidden response.
func Forbidden(w http.ResponseWriter, r *http.Request, message ...string) {
	w.WriteHeader(http.StatusForbidden)
	err := render.RenderTemplate(w, r, ".errors/403", &render.Vars{
		Message: message[0],
	})
	if err != nil {
		log.Error(err.Error())
		w.Write([]byte("Unrecoverable template error for Forbidden()"))
	}
}

// Error sends an HTTP 500 Internal Server Error response.
func Error(w http.ResponseWriter, r *http.Request, message ...string) {
	w.WriteHeader(http.StatusInternalServerError)
	err := render.RenderTemplate(w, r, ".errors/500", &render.Vars{
		Message: message[0],
	})
	if err != nil {
		log.Error(err.Error())
		w.Write([]byte("Unrecoverable template error for Error()"))
	}
}

// BadRequest sends an HTTP 400 Bad Request.
func BadRequest(w http.ResponseWriter, r *http.Request, message ...string) {
	w.WriteHeader(http.StatusBadRequest)
	err := render.RenderTemplate(w, r, ".errors/400", &render.Vars{
		Message: message[0],
	})
	if err != nil {
		log.Error(err.Error())
		w.Write([]byte("Unrecoverable template error for BadRequest()"))
	}
}
