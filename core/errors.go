package core

import (
	"net/http"

	"github.com/kirsle/blog/core/internal/log"
	"github.com/kirsle/blog/core/internal/render"
	"github.com/kirsle/blog/core/internal/responses"
)

// registerErrors loads the error handlers into the responses subpackage.
func (b *Blog) registerErrors() {
	responses.NotFound = func(w http.ResponseWriter, r *http.Request, message string) {
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

	responses.Forbidden = func(w http.ResponseWriter, r *http.Request, message string) {
		w.WriteHeader(http.StatusForbidden)
		err := render.Template(w, r, ".errors/403", map[string]string{
			"Message": message,
		})
		if err != nil {
			log.Error(err.Error())
			w.Write([]byte("Unrecoverable template error for Forbidden()"))
		}
	}

	responses.Error = func(w http.ResponseWriter, r *http.Request, message string) {
		w.WriteHeader(http.StatusInternalServerError)
		err := render.Template(w, r, ".errors/500", map[string]string{
			"Message": message,
		})
		if err != nil {
			log.Error(err.Error())
			w.Write([]byte("Unrecoverable template error for Error()"))
		}
	}

	responses.BadRequest = func(w http.ResponseWriter, r *http.Request, message string) {
		w.WriteHeader(http.StatusBadRequest)
		err := render.Template(w, r, ".errors/400", map[string]string{
			"Message": message,
		})
		if err != nil {
			log.Error(err.Error())
			w.Write([]byte("Unrecoverable template error for BadRequest()"))
		}
	}

}
