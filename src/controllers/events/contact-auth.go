package events

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/gorilla/mux"
	"github.com/kirsle/blog/src/log"
	"github.com/kirsle/blog/src/responses"
	"github.com/kirsle/blog/src/sessions"
	"github.com/kirsle/blog/models/contacts"
	"github.com/kirsle/blog/models/events"
)

// AuthedContact returns the current authenticated Contact, if any, on the session.
func AuthedContact(r *http.Request) (contacts.Contact, error) {
	session := sessions.Get(r)
	if contactID, ok := session.Values["contact-id"].(int); ok && contactID != 0 {
		contact, err := contacts.Get(contactID)
		return contact, err
	}
	return contacts.Contact{}, errors.New("not authenticated")
}

// contactAuthHandler listens at "/c/<contact secret>?e=<event id>"
//
// It is used in RSVP invite emails so when the user clicks the link, it auto
// authenticates their session as the contact ID using the contact secret
// (a randomly generated string in the DB). The ?e= param indicates an event
// ID to redirect to.
func contactAuthHandler(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	secret, ok := params["secret"]
	if !ok {
		responses.BadRequest(w, r, "Bad Request")
	}

	c, err := contacts.GetBySecret(secret)
	if err != nil {
		log.Error("contactAuthHandler (/c/<secret>): secret not found, don't know this user")
		responses.Redirect(w, "/")
		return
	}

	log.Info("contactAuthHandler: Contact %d (%s) is now authenticated", c.ID, c.Name())

	// Authenticate the contact in the session.
	session := sessions.Get(r)
	session.Values["contact-id"] = c.ID
	session.Values["c.name"] = c.Name() // comment form values auto-filled nicely
	session.Values["c.email"] = c.Email
	err = session.Save(r, w)
	if err != nil {
		log.Error("contactAuthHandler: save session error: %s", err)
	}

	// Did they give us an event ID?
	eventIDStr := r.FormValue("e")
	if eventIDStr != "" {
		if eventID, err := strconv.Atoi(eventIDStr); err == nil {
			event, err := events.Load(eventID)
			if err != nil {
				responses.FlashAndRedirect(w, r, "/", "Event %d not found", eventID)
				return
			}

			// Redirect to the event.
			responses.Redirect(w, "/e/"+event.Fragment)
			return
		}
	}

	// Redirect home I guess?
	log.Error("contactAuthHandler: don't know where to send them (no ?e= param for event ID)")
	responses.Redirect(w, "/")
}

func contactLogoutHandler(w http.ResponseWriter, r *http.Request) {
	session := sessions.Get(r)
	delete(session.Values, "contact-id")
	session.Save(r, w)

	if next := r.FormValue("next"); next != "" && strings.HasPrefix(next, "/") {
		responses.Redirect(w, next)
	} else {
		responses.Redirect(w, "/")
	}
}
