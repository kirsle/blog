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

	// Get the address book.
	addr, _ := contacts.Load()

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

			addr.Add(c)
			err = addr.Save()
			if err != nil {
				responses.FlashAndReload(w, r, "Error when saving address book: %s", err)
				return
			}

			responses.FlashAndReload(w, r, "Added %s to the address book!", c.Name())
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
				err := event.InviteContactID(id)
				if err != nil {
					warnings = append(warnings, err.Error())
				}
			}
			if len(warnings) > 0 {
				responses.Flash(w, r, "Warnings: %s", strings.Join(warnings, "; "))
			}
			responses.FlashAndReload(w, r, "Invites sent!")
			return
		}
	}

	invited, err := event.Invited()
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

	v := map[string]interface{}{
		"event":      event,
		"invited":    invited,
		"invitedMap": invitedMap,
		"contacts":   addr,
	}
	render.Template(w, r, "events/invite", v)
}
