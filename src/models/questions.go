package models

import (
	"errors"
	"net/http"
	"net/mail"
	"strconv"
	"time"
)

// Question is a question asked of the blog owner.
type Question struct {
	ID       int       `json:"id"`
	Name     string    `json:"name"`
	Email    string    `json:"email"`
	Question string    `json:"question"`
	Status   Status    `json:"status"`
	Created  time.Time `json:"created"`
	Updated  time.Time `json:"updated"`
}

// NewQuestion creates a blank Question with sensible defaults.
func NewQuestion() *Question {
	return &Question{
		Status: Pending,
	}
}

// GetQuestion by its ID.
func GetQuestion(id int) (*Question, error) {
	result := &Question{}
	err := DB.First(&result, id).Error
	return result, err
}

// AllQuestions returns all the Questions.
func AllQuestions() ([]*Question, error) {
	result := []*Question{}
	err := DB.Order("created desc").Find(&result).Error
	return result, err
}

// PendingQuestions returns pending questions in order of recency.
func PendingQuestions(offset, limit int) ([]*Question, error) {
	result := []*Question{}
	err := DB.Where("status = ?", Pending).
		Offset(offset).Limit(limit).
		Order("created desc").
		Find(&result).Error
	return result, err
}

// ParseForm populates the Question from form values.
func (ev *Question) ParseForm(r *http.Request) {
	id, _ := strconv.Atoi(r.FormValue("id"))

	ev.ID = id
	ev.Name = r.FormValue("name")
	ev.Email = r.FormValue("email")
	ev.Question = r.FormValue("question")
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
func (ev *Question) Validate() error {
	if ev.Question == "" {
		return errors.New("question is required")
	}
	if ev.Email != "" {
		if _, err := mail.ParseAddress(ev.Email); err != nil {
			return err
		}
	}
	return nil
}

// Load an Question by its ID.
func Load(id int) (*Question, error) {
	ev := &Question{}
	err := DB.First(ev, id).Error
	return ev, err
}

// Save the Question.
func (ev *Question) Save() error {
	if ev.Name == "" {
		ev.Name = "Anonymous"
	}

	// Dates & times.
	if ev.Created.IsZero() {
		ev.Created = time.Now().UTC()
	}
	if ev.Updated.IsZero() {
		ev.Updated = ev.Created
	}

	// Write the Question.
	return DB.Save(&ev).Error
}

// Delete an Question.
func (ev *Question) Delete() error {
	if ev.ID == 0 {
		return errors.New("Question has no ID")
	}

	// Delete the DB files.
	return DB.Delete(ev).Error
}
