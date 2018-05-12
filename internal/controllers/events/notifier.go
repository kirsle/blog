package events

import (
	"fmt"
	"strings"

	"github.com/kirsle/blog/internal/mail"
	"github.com/kirsle/blog/models/events"
	"github.com/kirsle/blog/models/settings"
)

func notifyUser(ev *events.Event, rsvp events.RSVP) {
	var (
		email = rsvp.GetEmail()
		sms   = rsvp.GetSMS()
	)
	s, _ := settings.Load()

	// Do they have... an e-mail address?
	if email != "" {
		mail.SendEmail(mail.Email{
			To:      email,
			Subject: fmt.Sprintf("Invitation to: %s", ev.Title),
			Data: map[string]interface{}{
				"RSVP":  rsvp,
				"Event": ev,
				"URL":   strings.Trim(s.Site.URL, "/") + "/e/" + ev.Fragment,
			},
			Template: ".email/event-invite.gohtml",
		})
	}

	// An SMS number?
	if sms != "" {

	}

	rsvp.Notified = true
	rsvp.Save()
}
