package core

import (
	"context"
	"net/http"

	"github.com/kirsle/blog/core/models/users"
)

// AuthMiddleware loads the user's authentication state.
func (b *Blog) AuthMiddleware(w http.ResponseWriter, r *http.Request, next http.HandlerFunc) {
	session, _ := b.store.Get(r, "session")
	log.Info("Session: %v", session.Values)
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
	}
	next(w, r)
}

// LoginRequired is a middleware that requires a logged-in user.
func (b *Blog) LoginRequired(w http.ResponseWriter, r *http.Request, next http.HandlerFunc) {
	ctx := r.Context()
	if user, ok := ctx.Value(userKey).(*users.User); ok {
		if user.ID > 0 {
			next(w, r)
		}
	}

	b.Redirect(w, "/login?next="+r.URL.Path)
}
