package types

// Key is an integer enum for context.Context keys.
type Key int

// Key definitions.
const (
	SessionKey   Key = iota // The request's cookie session object.
	UserKey                 // The request's user data for logged-in users.
	StartTimeKey            // HTTP request start time.
)
