package setup

import (
	"net/http"

	"github.com/gorilla/mux"
	"github.com/kirsle/blog/core/internal/forms"
	"github.com/kirsle/blog/core/internal/log"
	"github.com/kirsle/blog/core/internal/middleware/auth"
	"github.com/kirsle/blog/core/internal/models/settings"
	"github.com/kirsle/blog/core/internal/models/users"
	"github.com/kirsle/blog/core/internal/render"
	"github.com/kirsle/blog/core/internal/responses"
	"github.com/kirsle/blog/core/internal/sessions"
)

// Register the initial setup routes.
func Register(r *mux.Router) {
	r.HandleFunc("/initial-setup", handler)
}

func handler(w http.ResponseWriter, r *http.Request) {
	form := &forms.Setup{}
	vars := map[string]interface{}{
		"Form": form,
	}

	// Reject if we're already set up.
	s, _ := settings.Load()
	if s.Initialized {
		responses.FlashAndRedirect(w, r, "/", "This website has already been configured.")
		return
	}

	if r.Method == http.MethodPost {
		form.ParseForm(r)
		err := form.Validate()
		if err != nil {
			vars["Error"] = err
		} else {
			// Save the site config.
			log.Info("Creating default website config file")
			s := settings.Defaults()
			s.Save()

			// Re-initialize the cookie store with the new secret key.
			sessions.SetSecretKey([]byte(s.Security.SecretKey))

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
				vars["Error"] = err
			}

			// All set!
			auth.Login(w, r, user)
			responses.FlashAndRedirect(w, r, "/admin", "Admin user created and logged in.")
			return
		}
	}

	render.Template(w, r, "initial-setup", vars)
}
