package core

import (
	"fmt"
	"net/http"

	"github.com/kirsle/blog/core/models/users"
)

// SetupHandler is the initial blog setup route.
func (b *Blog) SetupHandler(w http.ResponseWriter, r *http.Request) {
	vars := map[string]interface{}{
		"errors": []error{},
	}

	if r.Method == "POST" {
		var errors []error
		payload := struct {
			Username string
			Password string
			Confirm  string
		}{
			Username: r.FormValue("username"),
			Password: r.FormValue("password"),
			Confirm:  r.FormValue("confirm"),
		}

		// Validate stuff.
		if len(payload.Username) == 0 {
			errors = append(errors, fmt.Errorf("Admin Username is required"))
		}
		if len(payload.Password) < 3 {
			errors = append(errors, fmt.Errorf("Admin Password is too short"))
		}
		if payload.Password != payload.Confirm {
			errors = append(errors, fmt.Errorf("Your passwords do not match"))
		}

		vars["errors"] = errors

		// No problems?
		if len(errors) == 0 {
			log.Info("Creating admin account %s", payload.Username)
			user := &users.User{
				Username: payload.Username,
				Password: payload.Password,
			}
			err := b.DB.Commit("users/by-name/"+payload.Username, user)
			if err != nil {
				log.Error("Error: %v", err)
				b.BadRequest(w, r, "DB error when writing user")
			}
		}
	}

	b.RenderTemplate(w, r, "admin/setup", vars)
}
