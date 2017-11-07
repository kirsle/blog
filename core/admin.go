package core

import (
	"net/http"

	"github.com/kirsle/blog/core/forms"
	"github.com/kirsle/blog/core/models/users"
)

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
			b.Redirect(w, "/admin")
			return
		}
	}

	b.RenderTemplate(w, r, "admin/setup", vars)
}
