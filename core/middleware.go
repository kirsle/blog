package core

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/sessions"
	"github.com/kirsle/blog/core/internal/log"
	"github.com/kirsle/blog/core/internal/models/users"
)

type key int

const (
	sessionKey key = iota
	userKey
	requestTimeKey
)

// SessionLoader gets the Gorilla session store and makes it available on the
// Request context.
//
// SessionLoader is the first custom middleware applied, so it takes the current
// datetime to make available later in the request and stores it on the request
// context.
func (b *Blog) SessionLoader(w http.ResponseWriter, r *http.Request, next http.HandlerFunc) {
	// Store the current datetime on the request context.
	ctx := context.WithValue(r.Context(), requestTimeKey, time.Now())

	// Get the Gorilla session and make it available in the request context.
	session, _ := b.store.Get(r, "session")
	ctx = context.WithValue(ctx, sessionKey, session)

	next(w, r.WithContext(ctx))
}

// Session returns the current request's session.
func (b *Blog) Session(r *http.Request) *sessions.Session {
	if r == nil {
		panic("Session(*http.Request) with a nil argument!?")
	}

	ctx := r.Context()
	if session, ok := ctx.Value(sessionKey).(*sessions.Session); ok {
		return session
	}

	log.Error(
		"Session(): didn't find session in request context! Getting it " +
			"from the session store instead.",
	)
	session, _ := b.store.Get(r, "session")
	return session
}

// CSRFMiddleware enforces CSRF tokens on all POST requests.
func (b *Blog) CSRFMiddleware(w http.ResponseWriter, r *http.Request, next http.HandlerFunc) {
	if r.Method == "POST" {
		session := b.Session(r)
		token := b.GenerateCSRFToken(w, r, session)
		if token != r.FormValue("_csrf") {
			log.Error("CSRF Mismatch: expected %s, got %s", r.FormValue("_csrf"), token)
			b.Forbidden(w, r, "Failed to validate CSRF token. Please try your request again.")
			return
		}
	}

	next(w, r)
}

// GenerateCSRFToken generates a CSRF token for the user and puts it in their session.
func (b *Blog) GenerateCSRFToken(w http.ResponseWriter, r *http.Request, session *sessions.Session) string {
	token, ok := session.Values["csrf"].(string)
	if !ok {
		token := uuid.New()
		session.Values["csrf"] = token.String()
		session.Save(r, w)
	}
	return token
}

// CurrentUser returns the current user's object.
func (b *Blog) CurrentUser(r *http.Request) (*users.User, error) {
	session := b.Session(r)
	if loggedIn, ok := session.Values["logged-in"].(bool); ok && loggedIn {
		id := session.Values["user-id"].(int)
		u, err := users.LoadReadonly(id)
		u.IsAuthenticated = true
		return u, err
	}

	return &users.User{
		Admin: false,
	}, errors.New("not authenticated")
}

// LoggedIn returns whether the current user is logged in to an account.
func (b *Blog) LoggedIn(r *http.Request) bool {
	session := b.Session(r)
	if loggedIn, ok := session.Values["logged-in"].(bool); ok && loggedIn {
		return true
	}
	return false
}

// AuthMiddleware loads the user's authentication state.
func (b *Blog) AuthMiddleware(w http.ResponseWriter, r *http.Request, next http.HandlerFunc) {
	u, err := b.CurrentUser(r)
	if err != nil {
		next(w, r)
		return
	}

	ctx := context.WithValue(r.Context(), userKey, u)
	next(w, r.WithContext(ctx))
}

// LoginRequired is a middleware that requires a logged-in user.
func (b *Blog) LoginRequired(w http.ResponseWriter, r *http.Request, next http.HandlerFunc) {
	ctx := r.Context()
	if user, ok := ctx.Value(userKey).(*users.User); ok {
		if user.ID > 0 {
			next(w, r)
			return
		}
	}

	log.Info("Redirect away!")
	b.Redirect(w, "/login?next="+r.URL.Path)
}
