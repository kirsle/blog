package forms

import "net/http"

// Form is an interface for forms that can validate themselves.
type Form interface {
	Parse(r *http.Request)
	Validate() error
}
