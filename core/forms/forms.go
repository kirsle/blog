package forms

// Form is an interface for forms that can validate themselves.
type Form interface {
	Validate() error
}
