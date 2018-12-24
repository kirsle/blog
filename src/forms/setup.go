package forms

import (
	"errors"
	"net/http"
)

// Setup is for the initial blog setup page at /initial-setup.
type Setup struct {
	Username string
	Password string
	Confirm  string
}

// Parse form values.
func (f *Setup) ParseForm(r *http.Request) {
	f.Username = r.FormValue("username")
	f.Password = r.FormValue("password")
	f.Confirm = r.FormValue("confirm")
}

// Validate the form.
func (f Setup) Validate() error {
	if len(f.Username) == 0 {
		return errors.New("admin username is required")
	} else if len(f.Password) == 0 {
		return errors.New("admin password is required")
	} else if f.Password != f.Confirm {
		return errors.New("your passwords do not match")
	}
	return nil
}
