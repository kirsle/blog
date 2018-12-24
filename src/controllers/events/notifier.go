package events

import (
	"fmt"
	"strings"

	"github.com/kirsle/blog/src/mail"
	"github.com/kirsle/blog/models/events"
	"github.com/kirsle/blog/models/settings"
)

func notifyUser(ev *events.Event, rsvp events.RSVP) {
	var (
		email = rsvp.GetEmail()
		sms   = rsvp.GetSMS()
	)
	s, _ := settings.Load()

	// Can we get an "auto-login" link?
	var claimURL string
	if rsvp.Contact.Secret != "" {
		claimURL = fmt.Sprintf("%s/c/%s?e=%d",
			strings.Trim(s.Site.URL, "/"),
			rsvp.Contact.Secret,
			ev.ID,
		)
	}

	// Do they have... an e-mail address?
	if email != "" {
		go mail.SendEmail(mail.Email{
			To:      email,
			Subject: fmt.Sprintf("Invitation to: %s", ev.Title),
			Data: map[string]interface{}{
				"RSVP":     rsvp,
				"Event":    ev,
				"URL":      strings.Trim(s.Site.URL, "/") + "/e/" + ev.Fragment,
				"ClaimURL": claimURL,
			},
			Template: ".email/event-invite.gohtml",
		})
	}

	// An SMS number?
	if sms != "" {
		// TODO: Twilio
	}

	rsvp.Notified = true
	rsvp.Save()
}
