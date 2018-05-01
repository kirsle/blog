package events

import (
	"errors"
	"fmt"
	"time"

	"github.com/kirsle/blog/models/contacts"
)

// RSVP status constants.
const (
	StatusInvited  = "invited"
	StatusGoing    = "going"
	StatusMaybe    = "maybe"
	StatusNotGoing = "not going"
)

// RSVP tracks invitations and confirmations to events.
type RSVP struct {
	// If the user was invited by an admin, they will have a ContactID and
	// not much else. Users who signed up themselves from an OpenSignup event
	// will have the metadata filled in instead.
	ContactID int               `json:"contactId"`
	Contact   *contacts.Contact `json:"-"`      // rel table not serialized to JSON
	Status    string            `json:"status"` // invited, going, maybe, not going
	Notified  bool              `json:"notified"`
	Name      string            `json:"name,omitempty"`
	Email     string            `json:"email,omitempty"`
	SMS       string            `json:"sms,omitempty"`
	Created   time.Time         `json:"created"`
	Updated   time.Time         `json:"updated"`
}

// InviteContactID enters an invitation for a contact ID.
func (ev *Event) InviteContactID(id int) error {
	// Make sure the ID isn't already in the list.
	for _, rsvp := range ev.RSVP {
		if rsvp.ContactID != 0 && rsvp.ContactID == id {
			return errors.New("already invited")
		}
	}

	ev.RSVP = append(ev.RSVP, RSVP{
		ContactID: id,
		Status:    StatusInvited,
		Created:   time.Now().UTC(),
		Updated:   time.Now().UTC(),
	})
	return ev.Save()
}

// Invited returns the RSVPs with Contact objects injected for contacts.
func (ev *Event) Invited() ([]RSVP, error) {
	cl, _ := contacts.Load()
	result := []RSVP{}
	for _, rsvp := range ev.RSVP {
		if rsvp.ContactID != 0 {
			fmt.Printf("cid: %d\n", rsvp.ContactID)
			c, err := cl.GetID(rsvp.ContactID)
			if err != nil {
				fmt.Printf("event.Invited error: %s", err)
			}
			rsvp.Contact = c
		}
		result = append(result, rsvp)
	}

	return result, nil
}
