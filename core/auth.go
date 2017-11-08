package core

import (
	"errors"
	"net/http"

	"github.com/kirsle/blog/core/forms"
	"github.com/kirsle/blog/core/models/users"
)

type key int

const (
	userKey key = iota
)

// Login logs the browser in as the given user.
func (b *Blog) Login(w http.ResponseWriter, r *http.Request, u *users.User) error {
	session, err := b.store.Get(r, "session") // TODO session name
	if err != nil {
		return err
	}
	session.Values["logged-in"] = true
	session.Values["user-id"] = u.ID
	session.Save(r, w)
	return nil
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
				b.Login(w, r, user)

				// A next URL given? TODO: actually get to work
				next := r.FormValue("next")
				log.Info("Redirect after login to: %s", next)
				if len(next) > 0 && next[0] == '/' {
					b.Redirect(w, next)
				} else {
					b.Redirect(w, "/")
				}
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
