package forms

import (
	"errors"
	"net/http"
)

// Contact form for the site admin.
type Contact struct {
	Name    string
	Email   string
	Subject string
	Message string
	Trap1   string // 'contact'
	Trap2   string // 'website'
}

// ParseForm parses the form.
func (c *Contact) ParseForm(r *http.Request) {
	c.Name = r.FormValue("name")
	c.Email = r.FormValue("email")
	c.Subject = r.FormValue("subject")
	c.Message = r.FormValue("message")
	c.Trap1 = r.FormValue("contact")
	c.Trap2 = r.FormValue("website")

	// Default values.
	if c.Name == "" {
		c.Name = "Anonymous"
	}
	if c.Subject == "" {
		c.Subject = "No Subject"
	}
}

// Validate the form.
func (c Contact) Validate() error {
	if len(c.Message) == 0 {
		return errors.New("message is required")
	}
	return nil
}
