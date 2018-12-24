package models

// Status of a Question.
type Status string

// Status options.
const (
	Pending  = "pending"
	Answered = "answered"
	Deleted  = "deleted"
)
