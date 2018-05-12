package events

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gorilla/mux"
	"github.com/kirsle/blog/internal/log"
	"github.com/kirsle/blog/internal/render"
	"github.com/kirsle/blog/internal/responses"
	"github.com/kirsle/blog/models/contacts"
	"github.com/kirsle/blog/models/events"
)

func inviteHandler(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	id, err := strconv.Atoi(params["id"])
	if err != nil {
		responses.FlashAndRedirect(w, r, "/e/admin/", "Invalid ID")
		return
	}

	// Load the event from its ID.
	event, err := events.Load(id)
	if err != nil {
		responses.FlashAndRedirect(w, r, "/e/admin/", "Can't load event: %s", err)
		return
	}

	// Handle POST requests.
	if r.Method == http.MethodPost {
		action := r.FormValue("action")

		switch action {
		case "new-contact":
			c := contacts.NewContact()
			c.ParseForm(r)
			err = c.Validate()
			if err != nil {
				responses.FlashAndReload(w, r, "Validation error: %s", err)
				return
			}

			err = contacts.Add(&c)
			if err != nil {
				responses.FlashAndReload(w, r, "Error when saving address book: %s", err)
				return
			}

			err = event.InviteContactID(c.ID)
			if err != nil {
				responses.Flash(w, r, "Error: couldn't invite contact: %s", err)
			}

			responses.FlashAndReload(w, r, "Added %s to the address book and added to invite list!", c.Name())
			return
		case "send-invite":
			log.Error("Send Invite!")
			r.ParseForm()
			contactIDs, ok := r.Form["invite"]
			if !ok {
				responses.Error(w, r, "Missing: invite (list of IDs)")
				return
			}

			// Invite all the users.
			var warnings []string
			for _, strID := range contactIDs {
				id, _ := strconv.Atoi(strID)
				err = event.InviteContactID(id)
				log.Debug("Inviting contact ID %d: err=%s", id, err)
				if err != nil {
					warnings = append(warnings, err.Error())
				}
			}
			if len(warnings) > 0 {
				responses.Flash(w, r, "Warnings: %s", strings.Join(warnings, "; "))
			}
			responses.FlashAndReload(w, r, "Invites sent!")
			return
		case "revoke-invite":
			idx, _ := strconv.Atoi(r.FormValue("index"))
			err := event.Uninvite(idx)
			if err != nil {
				responses.FlashAndReload(w, r, "Error deleting the invite: %s", err)
				return
			}
			responses.FlashAndReload(w, r, "Invite revoked!")
			return
		case "notify":
			// Notify all the invited users!
			for _, rsvp := range event.RSVP {
				if !rsvp.Notified || true {
					log.Info("Notify RSVP %s about Event %s", rsvp.GetName(), event.Title)
					notifyUser(event, rsvp)
				}
			}
			responses.FlashAndReload(w, r, "Notification emails and SMS messages sent out!")
			return
		}
	}

	invited := event.RSVP
	if err != nil {
		log.Error("error getting event.Invited: %s", err)
	}

	// Map the invited user IDs.
	invitedMap := map[int]bool{}
	for _, rsvp := range invited {
		if rsvp.ContactID != 0 {
			invitedMap[rsvp.ContactID] = true
		}
	}

	allContacts, err := contacts.All()
	if err != nil {
		log.Error("contacts.All() error: %s", err)
	}

	v := map[string]interface{}{
		"event":      event,
		"invited":    invited,
		"invitedMap": invitedMap,
		"contacts":   allContacts,
	}
	render.Template(w, r, "events/invite", v)
}
