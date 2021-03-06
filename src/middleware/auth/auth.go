package auth

import (
	"context"
	"errors"
	"net/http"

	"github.com/kirsle/blog/models/users"
	"github.com/kirsle/blog/src/sessions"
	"github.com/kirsle/blog/src/types"
	"github.com/urfave/negroni"
)

// CurrentUser returns the current user's object.
func CurrentUser(r *http.Request) (*users.User, error) {
	session := sessions.Get(r)
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

// Login logs the browser in as the given user.
func Login(w http.ResponseWriter, r *http.Request, u *users.User) error {
	session, err := sessions.Store.Get(r, "session") // TODO session name
	if err != nil {
		return err
	}
	session.Values["logged-in"] = true
	session.Values["user-id"] = u.ID
	session.Save(r, w)
	return nil
}

// LoggedIn returns whether the current user is logged in to an account.
func LoggedIn(r *http.Request) bool {
	session := sessions.Get(r)
	if loggedIn, ok := session.Values["logged-in"].(bool); ok && loggedIn {
		return true
	}
	return false
}

// LoginRequired is a middleware that requires a logged-in user.
func LoginRequired(onError http.HandlerFunc) negroni.HandlerFunc {
	middleware := func(w http.ResponseWriter, r *http.Request, next http.HandlerFunc) {
		ctx := r.Context()
		if user, ok := ctx.Value(types.UserKey).(*users.User); ok {
			if user.ID > 0 {
				next(w, r)
				return
			}
		}
		onError(w, r)
	}

	return middleware
}

// Middleware loads the user's authentication state from their session cookie.
func Middleware(w http.ResponseWriter, r *http.Request, next http.HandlerFunc) {
	u, err := CurrentUser(r)
	if err != nil {
		next(w, r)
		return
	}

	ctx := context.WithValue(r.Context(), types.UserKey, u)
	next(w, r.WithContext(ctx))
}
