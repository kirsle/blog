package core

// PostPrivacy values.
type PostPrivacy string

// ContentType values
type ContentType string

// Post privacy constants.
const (
	PUBLIC   PostPrivacy = "public"
	PRIVATE              = "private"
	UNLISTED             = "unlisted"
	DRAFT                = "draft"
)

// Content types for blog posts.
const (
	MARKDOWN ContentType = "markdown"
	HTML     ContentType = "html"
)

// Common form actions.
const (
	ActionSave    = "save"
	ActionDelete  = "delete"
	ActionPreview = "preview"
	ActionPost    = "post"
)
