package posts

import (
	"errors"
	"net/http"
	"strconv"
	"strings"
)

// Post holds information for a blog post.
type Post struct {
	ID             int      `json:"id"`
	Title          string   `json:"title"`
	Fragment       string   `json:"fragment"`
	ContentType    string   `json:"contentType"`
	Body           string   `json:"body"`
	Privacy        string   `json:"privacy"`
	Sticky         bool     `json:"sticky"`
	EnableComments bool     `json:"enableComments"`
	Tags           []string `json:"tags"`
}

// New creates a blank post with sensible defaults.
func New() *Post {
	return &Post{
		ContentType:    "markdown",
		Privacy:        "public",
		EnableComments: true,
	}
}

// LoadForm populates the post from form values.
func (p *Post) LoadForm(r *http.Request) {
	id, _ := strconv.Atoi(r.FormValue("id"))

	p.ID = id
	p.Title = r.FormValue("title")
	p.Fragment = r.FormValue("fragment")
	p.ContentType = r.FormValue("content-type")
	p.Body = r.FormValue("body")
	p.Privacy = r.FormValue("privacy")
	p.Sticky = r.FormValue("sticky") == "true"
	p.EnableComments = r.FormValue("enable-comments") == "true"

	// Ingest the tags.
	tags := strings.Split(r.FormValue("tags"), ",")
	p.Tags = []string{}
	for _, tag := range tags {
		p.Tags = append(p.Tags, strings.TrimSpace(tag))
	}
}

// Validate makes sure the required fields are all present.
func (p *Post) Validate() error {
	if p.Title == "" {
		return errors.New("title is required")
	}
	if p.ContentType != "markdown" && p.ContentType != "markdown+html" &&
		p.ContentType != "html" {
		return errors.New("invalid setting for ContentType")
	}
	if p.Privacy != "public" && p.Privacy != "draft" && p.Privacy != "private" {
		return errors.New("invalid setting for Privacy")
	}
	return nil
}
