package authctl

import (
	"errors"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/kirsle/blog/core/internal/forms"
	"github.com/kirsle/blog/core/internal/log"
	"github.com/kirsle/blog/core/internal/middleware/auth"
	"github.com/kirsle/blog/core/internal/models/users"
	"github.com/kirsle/blog/core/internal/render"
	"github.com/kirsle/blog/core/internal/responses"
	"github.com/kirsle/blog/core/internal/sessions"
)

// Register the initial setup routes.
func Register(r *mux.Router) {
	r.HandleFunc("/login", loginHandler)
	r.HandleFunc("/logout", logoutHandler)
	r.HandleFunc("/account", accountHandler)
}

func loginHandler(w http.ResponseWriter, r *http.Request) {
	vars := map[string]interface{}{
		"Form": forms.Setup{},
	}

	var nextURL string
	if r.Method == http.MethodPost {
		nextURL = r.FormValue("next")
	} else {
		nextURL = r.URL.Query().Get("next")
	}
	vars["NextURL"] = nextURL

	if r.Method == http.MethodPost {
		form := &forms.Login{
			Username: r.FormValue("username"),
			Password: r.FormValue("password"),
		}
		vars["Form"] = form
		err := form.Validate()
		if err != nil {
			vars["Error"] = err
		} else {
			// Test the login.
			user, err := users.CheckAuth(form.Username, form.Password)
			if err != nil {
				vars["Error"] = errors.New("bad username or password")
			} else {
				// Login OK!
				responses.Flash(w, r, "Login OK!")
				auth.Login(w, r, user)

				// A next URL given? TODO: actually get to work
				log.Info("Redirect after login to: %s", nextURL)
				if len(nextURL) > 0 && nextURL[0] == '/' {
					responses.Redirect(w, nextURL)
				} else {
					responses.Redirect(w, "/")
				}
				return
			}
		}
	}

	render.Template(w, r, "login", vars)
}

func logoutHandler(w http.ResponseWriter, r *http.Request) {
	session, _ := sessions.Store.Get(r, "session")
	delete(session.Values, "logged-in")
	delete(session.Values, "user-id")
	session.Save(r, w)
	responses.Redirect(w, "/")
}

func accountHandler(w http.ResponseWriter, r *http.Request) {
	if !auth.LoggedIn(r) {
		responses.FlashAndRedirect(w, r, "/login?next=/account", "You must be logged in to do that!")
		return
	}
	currentUser, err := auth.CurrentUser(r)
	if err != nil {
		responses.FlashAndRedirect(w, r, "/login?next=/account", "You must be logged in to do that!!")
		return
	}

	// Load an editable copy of the user.
	user, err := users.Load(currentUser.ID)
	if err != nil {
		responses.FlashAndRedirect(w, r, "/login?next=/account", "User ID %d not loadable?", currentUser.ID)
		return
	}

	form := &forms.Account{
		Username: user.Username,
		Email:    user.Email,
		Name:     user.Name,
	}
	v := map[string]interface{}{
		"Form": form,
	}

	if r.Method == http.MethodPost {
		form.Username = users.Normalize(r.FormValue("username"))
		form.Email = r.FormValue("email")
		form.Name = r.FormValue("name")
		form.OldPassword = r.FormValue("oldpassword")
		form.NewPassword = r.FormValue("newpassword")
		form.NewPassword2 = r.FormValue("newpassword2")
		if err = form.Validate(); err != nil {
			responses.Flash(w, r, err.Error())
		} else {
			var ok = true

			// Validate the username is available.
			if form.Username != user.Username {
				if _, err = users.LoadUsername(form.Username); err == nil {
					responses.Flash(w, r, "That username already exists.")
					ok = false
				}
			}

			// Changing their password?
			if len(form.OldPassword) > 0 {
				// Validate their old password.
				if _, err = users.CheckAuth(form.Username, form.OldPassword); err != nil {
					responses.Flash(w, r, "Your old password is incorrect.")
					ok = false
				} else {
					err = user.SetPassword(form.NewPassword)
					if err != nil {
						responses.Flash(w, r, "Change password error: %s", err)
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
					responses.Flash(w, r, "Error saving user: %s", err)
				} else {
					responses.FlashAndRedirect(w, r, "/account", "Settings saved!")
					return
				}
			}
		}
	}

	render.Template(w, r, "account", v)
}
