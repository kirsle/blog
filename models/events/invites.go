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
	ID        int              `json:"id"`
	ContactID int              `json:"contactId"`
	EventID   int              `json:"eventId"`
	Contact   contacts.Contact `json:"-" gorm:"save_associations:false"` // rel table not serialized to JSON
	Status    string           `json:"status"`                           // invited, going, maybe, not going
	Notified  bool             `json:"notified"`
	Name      string           `json:"name,omitempty"`
	Email     string           `json:"email,omitempty"`
	SMS       string           `json:"sms,omitempty"`
	Created   time.Time        `json:"created"`
	Updated   time.Time        `json:"updated"`
}

// GetName of the user in the RSVP (from the contact or the anonymous name).
func (r RSVP) GetName() string {
	if r.Contact.Name() != "" {
		return r.Contact.Name()
	}
	return r.Name
}

// GetEmail gets the user's email (from the contact or the anonymous email).
func (r RSVP) GetEmail() string {
	if r.Contact.Email != "" {
		return r.Contact.Email
	}
	return r.Email
}

// GetSMS gets the user's SMS number (from the contact or the anonymous sms).
func (r RSVP) GetSMS() string {
	if r.Contact.SMS != "" {
		return r.Contact.SMS
	}
	return r.SMS
}

// Save the RSVP.
func (r RSVP) Save() error {
	r.Updated = time.Now().UTC()
	return DB.Save(&r).Error
}

// InviteContactID enters an invitation for a contact ID.
func (ev *Event) InviteContactID(id int) error {
	// Make sure the ID isn't already in the list.
	for _, rsvp := range ev.RSVP {
		if rsvp.ContactID != 0 && rsvp.ContactID == id {
			return errors.New("already invited")
		}
	}

	rsvp := &RSVP{
		ContactID: id,
		EventID:   ev.ID,
		Status:    StatusInvited,
		Created:   time.Now().UTC(),
		Updated:   time.Now().UTC(),
	}
	return DB.Save(&rsvp).Error
}

// Uninvite removes an RSVP.
func (ev Event) Uninvite(id int) error {
	var rsvp RSVP
	err := DB.First(&rsvp, id).Error
	if err != nil {
		return err
	}

	fmt.Printf("UNIVNITE: we have rsvp=%+v", rsvp)
	return DB.Model(&ev).Association("RSVP").Delete(rsvp).Error
}
