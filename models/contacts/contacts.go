package contacts

import (
	"errors"
	"math/rand"
	"net/http"
	"strings"
	"time"

	"github.com/jinzhu/gorm"
	"github.com/kirsle/golog"
)

// DB is a reference to the parent app's gorm DB.
var DB *gorm.DB

// UseDB registers the DB from the root app.
func UseDB(db *gorm.DB) {
	DB = db
	DB.AutoMigrate(&Contact{})
}

var log *golog.Logger

func init() {
	log = golog.GetLogger("blog")
}

// Contact is an individual contact in the address book.
type Contact struct {
	ID        int       `json:"id"`
	Secret    string    `json:"secret" gorm:"unique"` // their lazy insecure login token
	FirstName string    `json:"firstName"`
	LastName  string    `json:"lastName"`
	Email     string    `json:"email"`
	SMS       string    `json:"sms"`
	LastSeen  time.Time `json:"lastSeen"`
	Created   time.Time `json:"created"`
	Updated   time.Time `json:"updated"`
}

// Contacts is the plurality of all contacts.
type Contacts []Contact

// NewContact initializes a new contact entry.
func NewContact() Contact {
	return Contact{}
}

// pre-save checks.
func (c *Contact) presave() {
	if c.Created.IsZero() {
		c.Created = time.Now().UTC()
	}
	if c.Updated.IsZero() {
		c.Updated = time.Now().UTC()
	}

	if c.Secret == "" {
		// Make a random ID.
		n := 8
		var letters = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")
		secret := make([]rune, n)
		for i := range secret {
			secret[i] = letters[rand.Intn(len(letters))]
		}
		c.Secret = string(secret)
	}
}

// Add a new contact.
func Add(c *Contact) error {
	c.presave()

	log.Error("contacts.Add: %+v", c)

	return DB.Create(&c).Error
}

// All contacts from the database alphabetically sorted.
func All() (Contacts, error) {
	result := Contacts{}
	err := DB.Order("last_name").Find(&result).Error
	return result, err
}

// Get a contact by ID.
func Get(id int) (Contact, error) {
	contact := Contact{}
	err := DB.First(&contact, id).Error
	return contact, err
}

// Save the contact.
func (c Contact) Save() error {
	c.presave()
	return DB.Update(&c).Error
}

// GetEmail queries a contact by email address.
func GetEmail(email string) (Contact, error) {
	contact := Contact{}
	err := DB.Where("email = ?", email).First(&contact).Error
	return contact, err
}

// GetSMS queries a contact by SMS number.
func GetSMS(number string) (Contact, error) {
	contact := Contact{}
	err := DB.Where("sms = ?", number).First(&contact).Error
	return contact, err
}

// Name returns a friendly name for the contact.
func (c Contact) Name() string {
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
func (c Contact) Validate() error {
	if c.Email == "" && c.SMS == "" {
		return errors.New("email or sms number required")
	}
	if c.FirstName == "" && c.LastName == "" {
		return errors.New("first or last name required")
	}

	// Check for uniqueness of email and SMS.
	if c.Email != "" {
		if _, err := GetEmail(c.Email); err == nil {
			return errors.New("email address already exists")
		}
	}
	if c.SMS != "" {
		if _, err := GetSMS(c.SMS); err == nil {
			return errors.New("sms number already exists")
		}
	}

	return nil
}
