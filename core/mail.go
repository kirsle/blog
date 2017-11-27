package core

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"html/template"
	"net/url"
	"strings"

	"github.com/kirsle/blog/core/models/comments"
	"github.com/kirsle/blog/core/models/settings"
	gomail "gopkg.in/gomail.v2"
)

// Email configuration.
type Email struct {
	To             string
	Admin          bool /* admin view of the email */
	Subject        string
	UnsubscribeURL string
	Data           map[string]interface{}

	Template string
}

// SendEmail sends an email.
func (b *Blog) SendEmail(email Email) {
	s, _ := settings.Load()
	if !s.Mail.Enabled || s.Mail.Host == "" || s.Mail.Port == 0 || s.Mail.Sender == "" {
		log.Info("Suppressing email: not completely configured")
		return
	}

	// Resolve the template.
	tmpl, err := b.ResolvePath(email.Template)
	if err != nil {
		log.Error("SendEmail: %s", err.Error())
		return
	}

	// Render the template to HTML.
	var html bytes.Buffer
	t := template.New(tmpl.Basename)
	t, err = template.ParseFiles(tmpl.Absolute)
	if err != nil {
		log.Error("SendEmail: template parsing error: %s", err.Error())
	}
	err = t.ExecuteTemplate(&html, tmpl.Basename, email)
	if err != nil {
		log.Error("SendEmail: template execution error: %s", err.Error())
	}

	m := gomail.NewMessage()
	m.SetHeader("From", s.Mail.Sender)
	m.SetHeader("To", email.To)
	m.SetHeader("Subject", email.Subject)
	m.SetBody("text/html", html.String())

	d := gomail.NewDialer(s.Mail.Host, s.Mail.Port, s.Mail.Username, s.Mail.Password)
	if b.Debug {
		d.TLSConfig = &tls.Config{
			InsecureSkipVerify: true,
		}
	}
	if err := d.DialAndSend(m); err != nil {
		log.Error("SendEmail: %s", err.Error())
	}
}

// NotifyComment sends notification emails about comments.
func (b *Blog) NotifyComment(c *comments.Comment) {
	s, _ := settings.Load()
	if s.Site.URL == "" {
		log.Error("Can't send comment notification because the site URL is not configured")
		return
	}

	// Prepare the email payload.
	email := Email{
		Template: ".email/comment.gohtml",
		Subject:  "Comment Added: " + c.Subject,
		Data: map[string]interface{}{
			"Name":    c.Name,
			"Subject": c.Subject,
			"Body":    template.HTML(b.RenderMarkdown(c.Body)),
			"URL":     strings.Trim(s.Site.URL, "/") + c.OriginURL,
			"QuickDelete": fmt.Sprintf("%s/comments/quick-delete?t=%s&d=%s",
				strings.Trim(s.Site.URL, "/"),
				url.QueryEscape(c.ThreadID),
				url.QueryEscape(c.DeleteToken),
			),
		},
	}

	// Email the site admins.
	config, _ := settings.Load()
	if config.Site.AdminEmail != "" {
		email.To = config.Site.AdminEmail
		email.Admin = true
		log.Info("Mail site admin '%s' about comment notification on '%s'", email.To, c.ThreadID)
		b.SendEmail(email)
	}

	// Email the subscribers.
	email.Admin = false
	m := comments.LoadMailingList()
	for _, to := range m.List(c.ThreadID) {
		if to == c.Email {
			continue // don't email yourself
		}
		email.To = to
		email.UnsubscribeURL = fmt.Sprintf("%s/comments/subscription?t=%s&e=%s",
			strings.Trim(s.Site.URL, "/"),
			url.QueryEscape(c.ThreadID),
			url.QueryEscape(to),
		)
		log.Info("Mail subscriber '%s' about comment notification on '%s'", email.To, c.ThreadID)
		b.SendEmail(email)
	}
}
