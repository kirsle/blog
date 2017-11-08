package core

import (
	"net/http"

	"github.com/gorilla/sessions"
	"github.com/kirsle/blog/core/forms"
	"github.com/kirsle/blog/core/models/settings"
	"github.com/kirsle/blog/core/models/users"
)

// AdminHandler is the admin landing page.
func (b *Blog) AdminHandler(w http.ResponseWriter, r *http.Request) {
	b.RenderTemplate(w, r, "admin/index", nil)
}

// SetupHandler is the initial blog setup route.
func (b *Blog) SetupHandler(w http.ResponseWriter, r *http.Request) {
	vars := &Vars{
		Form: forms.Setup{},
	}

	if r.Method == "POST" {
		form := forms.Setup{
			Username: r.FormValue("username"),
			Password: r.FormValue("password"),
			Confirm:  r.FormValue("confirm"),
		}
		vars.Form = form
		err := form.Validate()
		if err != nil {
			vars.Error = err
		} else {
			// Save the site config.
			log.Info("Creating default website config file")
			s := settings.Defaults()
			s.Save()

			// Re-initialize the cookie store with the new secret key.
			b.store = sessions.NewCookieStore([]byte(s.Security.SecretKey))

			log.Info("Creating admin account %s", form.Username)
			user := &users.User{
				Username: form.Username,
				Password: form.Password,
				Admin:    true,
				Name:     "Administrator",
			}
			err := users.Create(user)
			if err != nil {
				log.Error("Error: %v", err)
				vars.Error = err
			}

			// All set!
			b.Login(w, r, user)
			b.Redirect(w, "/admin")
			return
		}
	}

	b.RenderTemplate(w, r, "admin/setup", vars)
}
