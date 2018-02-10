package contact

import (
	"fmt"
	"html/template"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/gorilla/mux"
	"github.com/kirsle/blog/core/internal/forms"
	"github.com/kirsle/blog/core/internal/mail"
	"github.com/kirsle/blog/core/internal/markdown"
	"github.com/kirsle/blog/core/internal/models/settings"
	"github.com/kirsle/blog/core/internal/render"
	"github.com/kirsle/blog/core/internal/responses"
)

// Register attaches the contact URL to the app.
func Register(r *mux.Router) {
	r.HandleFunc("/contact", func(w http.ResponseWriter, r *http.Request) {
		form := &forms.Contact{}
		v := map[string]interface{}{
			"Form": form,
		}

		// If there is no site admin, show an error.
		cfg, err := settings.Load()
		if err != nil {
			responses.Error(w, r, "Error loading site configuration!")
			return
		} else if cfg.Site.AdminEmail == "" {
			responses.Error(w, r, "There is no admin email configured for this website!")
			return
		} else if !cfg.Mail.Enabled {
			responses.Error(w, r, "This website doesn't have an e-mail gateway configured.")
			return
		}

		// Posting?
		if r.Method == http.MethodPost {
			form.ParseForm(r)
			if err = form.Validate(); err != nil {
				responses.Flash(w, r, err.Error())
			} else {
				go mail.SendEmail(mail.Email{
					To:       cfg.Site.AdminEmail,
					Admin:    true,
					ReplyTo:  form.Email,
					Subject:  fmt.Sprintf("Contact Form on %s: %s", cfg.Site.Title, form.Subject),
					Template: ".email/contact.gohtml",
					Data: map[string]interface{}{
						"Name":    form.Name,
						"Message": template.HTML(markdown.RenderMarkdown(form.Message)),
						"Email":   form.Email,
					},
				})

				// Log it to disk, too.
				fh, err := os.OpenFile(filepath.Join(*render.UserRoot, ".contact.log"), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
				if err != nil {
					responses.Flash(w, r, "Error logging the message to disk: %s", err)
				} else {
					fh.WriteString(fmt.Sprintf(
						"Date: %s\nName: %s\nEmail: %s\nSubject: %s\n\n%s\n\n--------------------\n\n",
						time.Now().Format(time.UnixDate),
						form.Name,
						form.Email,
						form.Subject,
						form.Message,
					))
					fh.Close()
				}
				responses.FlashAndRedirect(w, r, "/contact", "Your message has been sent.")
			}
		}

		render.Template(w, r, "contact", v)
	})
}
