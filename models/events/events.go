package events

import (
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
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

// Load an event by its ID.
func Load(id int) (*Event, error) {
	ev := &Event{}
	err := DB.Get(fmt.Sprintf("events/by-id/%d", id), &ev)
	return ev, err
}

// LoadFragment loads an event by its URL fragment.
func LoadFragment(fragment string) (*Event, error) {
	idx, err := GetIndex()
	if err != nil {
		return nil, err
	}

	if id, ok := idx.Fragments[fragment]; ok {
		ev, err := Load(id)
		return ev, err
	}

	return nil, errors.New("fragment not found")
}

// Save the event.
func (ev *Event) Save() error {
	// Editing an existing event?
	if ev.ID == 0 {
		ev.ID = nextID()
	}

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
	DB.Commit(fmt.Sprintf("events/by-id/%d", ev.ID), ev)

	// Update the index cache.
	err := UpdateIndex(ev)
	if err != nil {
		return fmt.Errorf("UpdateIndex() error: %v", err)
	}

	return nil
}

// Delete an event.
func (ev *Event) Delete() error {
	if ev.ID == 0 {
		return errors.New("event has no ID")
	}

	// Delete the DB files.
	DB.Delete(fmt.Sprintf("events/by-id/%d", ev.ID))

	// Remove it from the index.
	idx, err := GetIndex()
	if err != nil {
		return fmt.Errorf("GetIndex error: %v", err)
	}
	return idx.Delete(ev)
}

// getNextID gets the next blog post ID.
func nextID() int {
	// Highest ID seen so far.
	var highest int

	events, err := DB.List("events/by-id")
	if err != nil {
		return 1
	}

	for _, doc := range events {
		fields := strings.Split(doc, "/")
		id, err := strconv.Atoi(fields[len(fields)-1])
		if err != nil {
			continue
		}

		if id > highest {
			highest = id
		}
	}

	// Return the highest +1
	return highest + 1
}
