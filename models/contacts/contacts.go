package contacts

import (
	"errors"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/kirsle/blog/jsondb"
	"github.com/kirsle/golog"
)

// DB is a reference to the parent app's JsonDB object.
var DB *jsondb.DB

var log *golog.Logger

func init() {
	log = golog.GetLogger("blog")
}

// Contacts is an address book of users who have been invited to events.
type Contacts struct {
	Serial   int        `json:"serial"`
	Contacts []*Contact `json:"contacts"`
}

// Contact is an individual contact in the address book.
type Contact struct {
	ID        int       `json:"id"`
	FirstName string    `json:"firstName"`
	LastName  string    `json:"lastName"`
	Email     string    `json:"email"`
	SMS       string    `json:"sms"`
	LastSeen  time.Time `json:"lastSeen"`
	Created   time.Time `json:"created"`
	Updated   time.Time `json:"updated"`
}

// NewContact initializes a new contact entry.
func NewContact() *Contact {
	return &Contact{}
}

// Load the singleton contact list.
func Load() (*Contacts, error) {
	c := &Contacts{
		Serial:   1,
		Contacts: []*Contact{},
	}
	if DB.Exists("contacts/address-book") {
		err := DB.Get("contacts/address-book", &c)
		return c, err
	}
	return c, nil
}

// Add a new contact.
func (cl *Contacts) Add(c *Contact) {
	if c.ID == 0 {
		c.ID = cl.Serial
		cl.Serial++
	}

	if c.Created.IsZero() {
		c.Created = time.Now().UTC()
	}
	if c.Updated.IsZero() {
		c.Updated = time.Now().UTC()
	}
	cl.Contacts = append(cl.Contacts, c)
}

// Save the contact list.
func (cl *Contacts) Save() error {
	sort.Sort(ByName(cl.Contacts))
	return DB.Commit("contacts/address-book", cl)
}

// GetID queries a contact by its ID number.
func (cl *Contacts) GetID(id int) (*Contact, error) {
	for _, c := range cl.Contacts {
		if c.ID == id {
			return c, nil
		}
	}
	return nil, errors.New("not found")
}

// GetEmail queries a contact by email address.
func (cl *Contacts) GetEmail(email string) (*Contact, error) {
	email = strings.ToLower(email)
	for _, c := range cl.Contacts {
		if c.Email == email {
			return c, nil
		}
	}
	return nil, errors.New("not found")
}

// GetSMS queries a contact by SMS number.
func (cl *Contacts) GetSMS(number string) (*Contact, error) {
	for _, c := range cl.Contacts {
		if c.SMS == number {
			return c, nil
		}
	}
	return nil, errors.New("not found")
}

// Name returns a friendly name for the contact.
func (c *Contact) Name() string {
	var parts []string
	if c.FirstName != "" {
		parts = append(parts, c.FirstName)
	}
	if c.LastName != "" {
		parts = append(parts, c.LastName)
	}
	if len(parts) == 0 {
		if c.Email != "" {
			parts = append(parts, c.Email)
		} else if c.SMS != "" {
			parts = append(parts, c.SMS)
		}
	}
	return strings.Join(parts, " ")
}

// ParseForm accepts form data for a contact.
func (c *Contact) ParseForm(r *http.Request) {
	c.FirstName = r.FormValue("first_name")
	c.LastName = r.FormValue("last_name")
	c.Email = strings.ToLower(r.FormValue("email"))
	c.SMS = r.FormValue("sms")
}

// Validate the contact form.
func (c *Contact) Validate() error {
	if c.Email == "" && c.SMS == "" {
		return errors.New("email or sms number required")
	}
	if c.FirstName == "" && c.LastName == "" {
		return errors.New("first or last name required")
	}

	// Get the address book out.
	addr, _ := Load()

	// Check for uniqueness of email and SMS.
	if c.Email != "" {
		if _, err := addr.GetEmail(c.Email); err == nil {
			return errors.New("email address already exists")
		}
	}
	if c.SMS != "" {
		if _, err := addr.GetSMS(c.SMS); err == nil {
			return errors.New("sms number already exists")
		}
	}

	return nil
}
