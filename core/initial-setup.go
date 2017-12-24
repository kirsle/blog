package core

import (
	"net/http"

	"github.com/gorilla/sessions"
	"github.com/kirsle/blog/core/forms"
	"github.com/kirsle/blog/core/models/settings"
	"github.com/kirsle/blog/core/models/users"
)

// SetupHandler is the initial blog setup route.
func (b *Blog) SetupHandler(w http.ResponseWriter, r *http.Request) {
	vars := &Vars{
		Form: forms.Setup{},
	}

	// Reject if we're already set up.
	s, _ := settings.Load()
	if s.Initialized {
		b.FlashAndRedirect(w, r, "/", "This website has already been configured.")
		return
	}

	if r.Method == http.MethodPost {
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
			b.FlashAndRedirect(w, r, "/admin", "Admin user created and logged in.")
			return
		}
	}

	b.RenderTemplate(w, r, "initial-setup", vars)
}
