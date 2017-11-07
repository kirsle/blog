package forms

import (
	"errors"
)

// Setup is for the initial blog setup page at /admin/setup.
type Setup struct {
	Username string
	Password string
	Confirm  string
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
