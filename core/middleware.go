package core

import (
	"context"
	"net/http"

	"github.com/gorilla/sessions"
	"github.com/kirsle/blog/core/models/users"
)

type key int

const (
	sessionKey key = iota
	userKey
)

// SessionLoader gets the Gorilla session store and makes it available on the
// Request context.
func (b *Blog) SessionLoader(w http.ResponseWriter, r *http.Request, next http.HandlerFunc) {
	session, _ := b.store.Get(r, "session")

	log.Debug("REQUEST START: %s %s", r.Method, r.URL.Path)
	ctx := context.WithValue(r.Context(), sessionKey, session)
	next(w, r.WithContext(ctx))
}

// Session returns the current request's session.
func (b *Blog) Session(r *http.Request) *sessions.Session {
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

// AuthMiddleware loads the user's authentication state.
func (b *Blog) AuthMiddleware(w http.ResponseWriter, r *http.Request, next http.HandlerFunc) {
	session := b.Session(r)
	log.Debug("AuthMiddleware() -- session values: %v", session.Values)
	if loggedIn, ok := session.Values["logged-in"].(bool); ok && loggedIn {
		// They seem to be logged in. Get their user object.
		id := session.Values["user-id"].(int)
		u, err := users.Load(id)
		if err != nil {
			log.Error("Error loading user ID %d from session: %v", id, err)
			next(w, r)
			return
		}

		ctx := context.WithValue(r.Context(), userKey, u)
		next(w, r.WithContext(ctx))
		return
	}
	next(w, r)
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
