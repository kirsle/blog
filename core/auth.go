package core

import (
	"context"
	"errors"
	"net/http"

	"github.com/kirsle/blog/core/forms"
	"github.com/kirsle/blog/core/models/users"
)

type key int

const (
	userKey key = iota
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

// LoginHandler shows and handles the login page.
func (b *Blog) LoginHandler(w http.ResponseWriter, r *http.Request) {
	vars := &Vars{
		Form: forms.Setup{},
	}

	if r.Method == "POST" {
		form := &forms.Login{
			Username: r.FormValue("username"),
			Password: r.FormValue("password"),
		}
		vars.Form = form
		err := form.Validate()
		if err != nil {
			vars.Error = err
		} else {
			// Test the login.
			user, err := users.CheckAuth(form.Username, form.Password)
			if err != nil {
				vars.Error = errors.New("bad username or password")
			} else {
				// Login OK!
				vars.Flash = "Login OK!"

				// Log in the user.
				session, err := b.store.Get(r, "session") // TODO session name
				if err != nil {
					vars.Error = err
				} else {
					session.Values["logged-in"] = true
					session.Values["user-id"] = user.ID
					session.Save(r, w)
				}

				b.Redirect(w, "/login")
				return
			}
		}
	}

	b.RenderTemplate(w, r, "login", vars)
}

// LogoutHandler logs the user out and redirects to the home page.
func (b *Blog) LogoutHandler(w http.ResponseWriter, r *http.Request) {
	session, _ := b.store.Get(r, "session")
	delete(session.Values, "logged-in")
	delete(session.Values, "user-id")
	session.Save(r, w)
	b.Redirect(w, "/")
}
