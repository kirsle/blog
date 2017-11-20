package forms

import (
	"errors"
	"net/mail"
)

// Settings are the user-facing admin settings.
type Settings struct {
	Title        string
	AdminEmail   string
	RedisEnabled bool
	RedisHost    string
	RedisPort    int
	RedisDB      int
	RedisPrefix  string
}

// Validate the form.
func (f Settings) Validate() error {
	if len(f.Title) == 0 {
		return errors.New("website title is required")
	}
	if f.AdminEmail != "" {
		_, err := mail.ParseAddress(f.AdminEmail)
		if err != nil {
			return err
		}
	}
	return nil
}
