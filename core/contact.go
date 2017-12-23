package core

import (
	"fmt"
	"html/template"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/kirsle/blog/core/forms"
	"github.com/kirsle/blog/core/models/settings"
)

// ContactRoutes attaches the contact URL to the app.
func (b *Blog) ContactRoutes(r *mux.Router) {
	r.HandleFunc("/contact", func(w http.ResponseWriter, r *http.Request) {
		v := NewVars()
		form := forms.Contact{}
		v.Form = &form

		// If there is no site admin, show an error.
		cfg, err := settings.Load()
		if err != nil {
			b.Error(w, r, "Error loading site configuration!")
			return
		} else if cfg.Site.AdminEmail == "" {
			b.Error(w, r, "There is no admin email configured for this website!")
			return
		} else if !cfg.Mail.Enabled {
			b.Error(w, r, "This website doesn't have an e-mail gateway configured.")
			return
		}

		// Posting?
		if r.Method == http.MethodPost {
			form.ParseForm(r)
			if err = form.Validate(); err != nil {
				b.Flash(w, r, err.Error())
			} else {
				go b.SendEmail(Email{
					To:       cfg.Site.AdminEmail,
					Admin:    true,
					ReplyTo:  form.Email,
					Subject:  fmt.Sprintf("Contact Form on %s: %s", cfg.Site.Title, form.Subject),
					Template: ".email/contact.gohtml",
					Data: map[string]interface{}{
						"Name":    form.Name,
						"Message": template.HTML(b.RenderMarkdown(form.Message)),
						"Email":   form.Email,
					},
				})
				b.FlashAndRedirect(w, r, "/contact", "Your message has been sent.")
			}
		}

		b.RenderTemplate(w, r, "contact", v)
	})
}
