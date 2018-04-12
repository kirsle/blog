package middleware

import (
	"net/http"

	"github.com/google/uuid"
	gorilla "github.com/gorilla/sessions"
	"github.com/kirsle/blog/internal/log"
	"github.com/kirsle/blog/internal/sessions"
	"github.com/urfave/negroni"
)

// CSRF is a middleware generator that enforces CSRF tokens on all POST requests.
func CSRF(onError func(http.ResponseWriter, *http.Request, string)) negroni.HandlerFunc {
	middleware := func(w http.ResponseWriter, r *http.Request, next http.HandlerFunc) {
		session := sessions.Get(r)
		token := GenerateCSRFToken(w, r, session)
		if r.Method == "POST" {
			if token != r.FormValue("_csrf") {
				log.Error("CSRF Mismatch: expected %s, got %s", r.FormValue("_csrf"), token)
				onError(w, r, "Failed to validate CSRF token. Please try your request again.")
				return
			}
		}
		next(w, r)
	}

	return middleware
}

// ExampleCSRF shows how to use the CSRF handler.
func ExampleCSRF() {
	// Your error handling for CSRF failures.
	onError := func(w http.ResponseWriter, r *http.Request, message string) {
		w.Write([]byte("CSRF Error: " + message))
	}

	// Attach the middleware.
	_ = negroni.New(
		negroni.NewRecovery(),
		negroni.NewLogger(),
		negroni.HandlerFunc(CSRF(onError)),
	)
}

// GenerateCSRFToken generates a CSRF token for the user and puts it in their session.
func GenerateCSRFToken(w http.ResponseWriter, r *http.Request, session *gorilla.Session) string {
	token, ok := session.Values["csrf"].(string)
	if !ok {
		token := uuid.New()
		session.Values["csrf"] = token.String()
		session.Save(r, w)
	}
	return token
}
