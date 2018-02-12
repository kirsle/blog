package responses

import (
	"fmt"
	"net/http"

	"github.com/kirsle/blog/internal/sessions"
)

// Error handlers to be filled in by the blog app.
var (
	NotFound   func(http.ResponseWriter, *http.Request, string)
	Forbidden  func(http.ResponseWriter, *http.Request, string)
	BadRequest func(http.ResponseWriter, *http.Request, string)
	Error      func(http.ResponseWriter, *http.Request, string)
)

// Flash adds a flash message to the user's session.
func Flash(w http.ResponseWriter, r *http.Request, message string, args ...interface{}) {
	session := sessions.Get(r)
	session.AddFlash(fmt.Sprintf(message, args...))
	session.Save(r, w)
}

// FlashAndRedirect flashes and redirects in one go.
func FlashAndRedirect(w http.ResponseWriter, r *http.Request, location, message string, args ...interface{}) {
	Flash(w, r, message, args...)
	Redirect(w, location)
}

// FlashAndReload flashes and sends a redirect to the same path.
func FlashAndReload(w http.ResponseWriter, r *http.Request, message string, args ...interface{}) {
	Flash(w, r, message, args...)
	Redirect(w, r.URL.Path)
}

// Redirect sends an HTTP redirect response.
func Redirect(w http.ResponseWriter, location string) {
	w.Header().Set("Location", location)
	w.WriteHeader(http.StatusFound)
}
