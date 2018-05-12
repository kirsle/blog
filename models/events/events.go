package events

import (
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/jinzhu/gorm"
	"github.com/kirsle/blog/models/contacts"
	"github.com/kirsle/golog"
)

// DB is a reference to the parent app's gorm DB.
var DB *gorm.DB

// UseDB registers the DB from the root app.
func UseDB(db *gorm.DB) {
	DB = db
	DB.AutoMigrate(&Event{}, &RSVP{})
	DB.Model(&Event{}).Related(&RSVP{})
	DB.Model(&RSVP{}).Related(&contacts.Contact{})
}

var log *golog.Logger

func init() {
	log = golog.GetLogger("blog")
}

// Event holds information about events.
type Event struct {
	ID          int       `json:"id"`
	Title       string    `json:"title"`
	Fragment    string    `json:"fragment"`
	Description string    `json:"description"`
	Location    string    `json:"location"`
	CoverPhoto  string    `json:"coverPhoto"`
	StartTime   time.Time `json:"startTime"`
	EndTime     time.Time `json:"endTime"`
	AllDay      bool      `json:"allDay"`
	OpenSignup  bool      `json:"openSignup"`
	RSVP        []RSVP    `json:"rsvp"`
	Created     time.Time `json:"created"`
	Updated     time.Time `json:"updated"`
}

// New creates a blank event with sensible defaults.
func New() *Event {
	return &Event{
		StartTime: time.Now().UTC(),
		EndTime:   time.Now().UTC(),
	}
}

// All returns all the events.
func All() ([]*Event, error) {
	result := []*Event{}
	err := DB.Order("start_time desc").Find(&result).Error
	return result, err
}

// ParseForm populates the event from form values.
func (ev *Event) ParseForm(r *http.Request) {
	id, _ := strconv.Atoi(r.FormValue("id"))

	ev.ID = id
	ev.Title = r.FormValue("title")
	ev.Fragment = r.FormValue("fragment")
	ev.Description = r.FormValue("description")
	ev.Location = r.FormValue("location")
	ev.AllDay = r.FormValue("all_day") == "true"
	ev.OpenSignup = r.FormValue("open_signup") == "true"

	startTime, err := parseDateTime(r, "start_date", "start_time")
	ev.StartTime = startTime
	if err != nil {
		log.Error("startTime parse error: %s", err)
	}

	endTime, err := parseDateTime(r, "end_date", "end_time")
	ev.EndTime = endTime
	if err != nil {
		log.Error("endTime parse error: %s", err)
	}
}

// parseDateTime parses separate date + time fields into a single time.Time.
func parseDateTime(r *http.Request, dateField, timeField string) (time.Time, error) {
	dateValue := r.FormValue(dateField)
	timeValue := r.FormValue(timeField)

	if dateValue != "" && timeValue != "" {
		datetime, err := time.Parse("2006-01-02 15:04", dateValue+" "+timeValue)
		return datetime, err
	} else if dateValue != "" {
		datetime, err := time.Parse("2006-01-02", dateValue)
		return datetime, err
	} else {
		return time.Time{}, errors.New("no date/times given")
	}
}

// Validate makes sure the required fields are all present.
func (ev *Event) Validate() error {
	if ev.Title == "" {
		return errors.New("title is required")
	} else if ev.Description == "" {
		return errors.New("description is required")
	}
	return nil
}

// joinedLoad loads the Event with its RSVPs and their Contacts.
func joinedLoad() *gorm.DB {
	return DB.Preload("RSVP").Preload("RSVP.Contact")
}

// Load an event by its ID.
func Load(id int) (*Event, error) {
	ev := &Event{}
	err := joinedLoad().First(ev, id).Error
	return ev, err
}

// LoadFragment loads an event by its URL fragment.
func LoadFragment(fragment string) (*Event, error) {
	ev := &Event{}
	err := joinedLoad().Where("fragment = ?", fragment).First(ev).Error
	return ev, err
}

// Save the event.
func (ev *Event) Save() error {
	// Generate a URL fragment if needed.
	if ev.Fragment == "" {
		fragment := strings.ToLower(ev.Title)
		fragment = regexp.MustCompile(`[^A-Za-z0-9]+`).ReplaceAllString(fragment, "-")
		if strings.Contains(fragment, "--") {
			log.Error("Generated event fragment '%s' contains double dashes still!", fragment)
		}
		ev.Fragment = strings.Trim(fragment, "-")

		// If still no fragment, make one based on the post ID.
		if ev.Fragment == "" {
			ev.Fragment = fmt.Sprintf("event-%d", ev.ID)
		}
	}

	// Make sure the URL fragment is unique!
	if len(ev.Fragment) > 0 {
		if exist, err := LoadFragment(ev.Fragment); err == nil && exist.ID != ev.ID {
			var resolved bool
			for i := 1; i <= 100; i++ {
				fragment := fmt.Sprintf("%s-%d", ev.Fragment, i)
				_, err := LoadFragment(fragment)
				if err == nil {
					continue
				}

				ev.Fragment = fragment
				resolved = true
				break
			}

			if !resolved {
				return fmt.Errorf("failed to generate a unique URL fragment for '%s' after 100 attempts", ev.Fragment)
			}
		}
	}

	// Dates & times.
	if ev.Created.IsZero() {
		ev.Created = time.Now().UTC()
	}
	if ev.Updated.IsZero() {
		ev.Updated = ev.Created
	}

	// Write the event.
	return DB.Save(&ev).Error
}

// Delete an event.
func (ev *Event) Delete() error {
	if ev.ID == 0 {
		return errors.New("event has no ID")
	}

	// Delete the DB files.
	return DB.Delete(ev).Error
}
