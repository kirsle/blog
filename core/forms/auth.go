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

// Account is for updating account settings.
type Account struct {
	Username     string
	OldPassword  string
	NewPassword  string
	NewPassword2 string
	Email        string
	Name         string
}

// Validate the account form.
func (f Account) Validate() error {
	if len(f.Username) == 0 {
		return errors.New("username is required")
	}
	if len(f.OldPassword) > 0 && len(f.NewPassword) > 0 {
		if f.NewPassword != f.NewPassword2 {
			return errors.New("your passwords don't match")
		}
	}
	if len(f.Name) == 0 {
		f.Name = f.Username
	}
	return nil
}
