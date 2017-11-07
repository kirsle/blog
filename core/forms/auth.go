package forms

import (
	"errors"
)

// Login is for signing into an account.
type Login struct {
	Username string
	Password string
}

// Validate the form.
func (f Login) Validate() error {
	if len(f.Username) == 0 {
		return errors.New("username is required")
	} else if len(f.Password) == 0 {
		return errors.New("password is required")
	}
	return nil
}
