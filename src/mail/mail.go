package mail

import (
	"bytes"
	"fmt"
	"html/template"
	"net/mail"
	"net/url"
	"strings"

	"github.com/kirsle/blog/src/log"
	"github.com/kirsle/blog/src/markdown"
	"github.com/kirsle/blog/src/render"
	"github.com/kirsle/blog/models/comments"
	"github.com/kirsle/blog/models/settings"
	"github.com/microcosm-cc/bluemonday"
	gomail "gopkg.in/gomail.v2"
)

// Email configuration.
type Email struct {
	To             string
	ReplyTo        string
	Admin          bool /* admin view of the email */
	Subject        string
	UnsubscribeURL string
	Data           map[string]interface{}

	Template string
}

// SendEmail sends an email.
func SendEmail(email Email) {
	s, _ := settings.Load()

	// Suppress sending any mail when no mail settings are configured, but go
	// through the motions -- great for local dev.
	var doNotMail bool
	if !s.Mail.Enabled || s.Mail.Host == "" || s.Mail.Port == 0 || s.Mail.Sender == "" {
		log.Info("Suppressing email: not completely configured")
		doNotMail = true
	}

	// Resolve the template.
	tmpl, err := render.ResolvePath(email.Template)
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

	// Condense the body down to plain text, lazily. Who even has a plain text
	// email client anymore?
	rawLines := strings.Split(
		bluemonday.StrictPolicy().Sanitize(html.String()),
		"\n",
	)
	var lines []string
	for _, line := range rawLines {
		line = strings.TrimSpace(line)
		if len(line) == 0 {
			continue
		}
		lines = append(lines, line)
	}
	plaintext := strings.Join(lines, "\n\n")

	// If we're not actually going to send the mail, this is a good place to stop.
	if doNotMail {
		log.Info("Not going to send an email.")
		log.Debug("The message was going to be:\n%s", plaintext)
		return
	}

	m := gomail.NewMessage()
	m.SetHeader("From", fmt.Sprintf("%s <%s>", s.Site.Title, s.Mail.Sender))
	m.SetHeader("To", email.To)
	if email.ReplyTo != "" {
		m.SetHeader("Reply-To", email.ReplyTo)
	}
	m.SetHeader("Subject", email.Subject)
	m.SetBody("text/plain", plaintext)
	m.AddAlternative("text/html", html.String())

	d := gomail.NewDialer(s.Mail.Host, s.Mail.Port, s.Mail.Username, s.Mail.Password)

	log.Info("SendEmail: %s (%s) to %s", email.Subject, email.Template, email.To)
	if err := d.DialAndSend(m); err != nil {
		log.Error("SendEmail: %s", err.Error())
	}
}

// NotifyComment sends notification emails about comments.
func NotifyComment(c *comments.Comment) {
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
			"Body":    template.HTML(markdown.RenderMarkdown(c.Body)),
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
		SendEmail(email)
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
		SendEmail(email)
	}
}

// ParseAddress parses an email address.
func ParseAddress(addr string) (*mail.Address, error) {
	return mail.ParseAddress(addr)
}
