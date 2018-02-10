package core

import (
	"errors"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/kirsle/blog/core/internal/forms"
	"github.com/kirsle/blog/core/internal/log"
	"github.com/kirsle/blog/core/internal/models/users"
)

// AuthRoutes attaches the auth routes to the app.
func (b *Blog) AuthRoutes(r *mux.Router) {
	r.HandleFunc("/login", b.LoginHandler)
	r.HandleFunc("/logout", b.LogoutHandler)
	r.HandleFunc("/account", b.AccountHandler)
}

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
	vars := NewVars()
	vars.Form = forms.Setup{}

	var nextURL string
	if r.Method == http.MethodPost {
		nextURL = r.FormValue("next")
	} else {
		nextURL = r.URL.Query().Get("next")
	}
	vars.Data["NextURL"] = nextURL

	if r.Method == http.MethodPost {
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
				b.Flash(w, r, "Login OK!")
				b.Login(w, r, user)

				// A next URL given? TODO: actually get to work
				log.Info("Redirect after login to: %s", nextURL)
				if len(nextURL) > 0 && nextURL[0] == '/' {
					b.Redirect(w, nextURL)
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

// AccountHandler shows the account settings page.
func (b *Blog) AccountHandler(w http.ResponseWriter, r *http.Request) {
	if !b.LoggedIn(r) {
		b.FlashAndRedirect(w, r, "/login?next=/account", "You must be logged in to do that!")
		return
	}
	currentUser, err := b.CurrentUser(r)
	if err != nil {
		b.FlashAndRedirect(w, r, "/login?next=/account", "You must be logged in to do that!!")
		return
	}

	// Load an editable copy of the user.
	user, err := users.Load(currentUser.ID)
	if err != nil {
		b.FlashAndRedirect(w, r, "/login?next=/account", "User ID %d not loadable?", currentUser.ID)
		return
	}

	v := NewVars()
	form := &forms.Account{
		Username: user.Username,
		Email:    user.Email,
		Name:     user.Name,
	}
	v.Form = form

	if r.Method == http.MethodPost {
		form.Username = users.Normalize(r.FormValue("username"))
		form.Email = r.FormValue("email")
		form.Name = r.FormValue("name")
		form.OldPassword = r.FormValue("oldpassword")
		form.NewPassword = r.FormValue("newpassword")
		form.NewPassword2 = r.FormValue("newpassword2")
		if err = form.Validate(); err != nil {
			b.Flash(w, r, err.Error())
		} else {
			var ok = true

			// Validate the username is available.
			if form.Username != user.Username {
				if _, err = users.LoadUsername(form.Username); err == nil {
					b.Flash(w, r, "That username already exists.")
					ok = false
				}
			}

			// Changing their password?
			if len(form.OldPassword) > 0 {
				// Validate their old password.
				if _, err = users.CheckAuth(form.Username, form.OldPassword); err != nil {
					b.Flash(w, r, "Your old password is incorrect.")
					ok = false
				} else {
					err = user.SetPassword(form.NewPassword)
					if err != nil {
						b.Flash(w, r, "Change password error: %s", err)
						ok = false
					}
				}
			}

			// Still good?
			if ok {
				user.Username = form.Username
				user.Name = form.Name
				user.Email = form.Email
				err = user.Save()
				if err != nil {
					b.Flash(w, r, "Error saving user: %s", err)
				} else {
					b.FlashAndRedirect(w, r, "/account", "Settings saved!")
					return
				}
			}
		}
	}

	b.RenderTemplate(w, r, "account", v)
}
