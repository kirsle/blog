package comments

import (
	"crypto/md5"
	"errors"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"net/mail"
	"time"

	"github.com/google/uuid"
	"github.com/kirsle/blog/core/jsondb"
	"github.com/kirsle/golog"
)

// DB is a reference to the parent app's JsonDB object.
var DB *jsondb.DB

var log *golog.Logger

func init() {
	log = golog.GetLogger("blog")
}

// Thread contains a thread of comments, for a blog post or otherwise.
type Thread struct {
	ID       string     `json:"id"`
	Comments []*Comment `json:"comments"`
}

// Comment contains the data for a single comment in a thread.
type Comment struct {
	ID          string    `json:"id"`
	UserID      int       `json:"userId,omitempty"`
	Name        string    `json:"name,omitempty"`
	Email       string    `json:"email,omitempty"`
	Avatar      string    `json:"avatar"`
	Body        string    `json:"body"`
	EditToken   string    `json:"editToken"`
	DeleteToken string    `json:"deleteToken"`
	Created     time.Time `json:"created"`
	Updated     time.Time `json:"updated"`

	// Private form use only.
	CSRF      string        `json:"-"`
	Subscribe bool          `json:"-"`
	ThreadID  string        `json:"-"`
	OriginURL string        `json:"-"`
	Subject   string        `json:"-"`
	HTML      template.HTML `json:"-"`
	Trap1     string        `json:"-"`
	Trap2     string        `json:"-"`

	// Even privater fields.
	IsAuthenticated bool   `json:"-"`
	Username        string `json:"-"`
	Editable        bool   `json:"-"`
	Editing         bool   `json:"-"`
}

// New initializes a new comment thread.
func New(id string) Thread {
	return Thread{
		ID:       id,
		Comments: []*Comment{},
	}
}

// Load a comment thread.
func Load(id string) (Thread, error) {
	t := Thread{}
	err := DB.Get(fmt.Sprintf("comments/threads/%s", id), &t)
	return t, err
}

// Post a comment to a thread.
func (t *Thread) Post(c *Comment) error {
	// If it has an ID, update an existing comment.
	if len(c.ID) > 0 {
		idx := -1
		for i, comment := range t.Comments {
			if comment.ID == c.ID {
				idx = i
				break
			}
		}

		// Replace the comment by index.
		if idx >= 0 && idx < len(t.Comments) {
			t.Comments[idx] = c
			DB.Commit(fmt.Sprintf("comments/threads/%s", t.ID), t)
			return nil
		}
	}

	// Assign an ID.
	if c.ID == "" {
		c.ID = uuid.New().String()
	}
	if c.DeleteToken == "" {
		c.DeleteToken = uuid.New().String()
	}

	t.Comments = append(t.Comments, c)
	DB.Commit(fmt.Sprintf("comments/threads/%s", t.ID), t)
	return nil
}

// Find a comment by its ID.
func (t *Thread) Find(id string) (*Comment, error) {
	for _, c := range t.Comments {
		if c.ID == id {
			return c, nil
		}
	}
	return nil, errors.New("comment not found")
}

// Delete a comment by its ID.
func (t *Thread) Delete(id string) error {
	keep := []*Comment{}
	var found bool
	for _, c := range t.Comments {
		if c.ID != id {
			keep = append(keep, c)
		} else {
			found = true
		}
	}

	if !found {
		return errors.New("comment not found")
	}

	t.Comments = keep
	DB.Commit(fmt.Sprintf("comments/threads/%s", t.ID), t)
	return nil
}

// FindByDeleteToken finds a comment by its deletion token.
func (t *Thread) FindByDeleteToken(token string) (*Comment, error) {
	for _, c := range t.Comments {
		if c.DeleteToken == token {
			return c, nil
		}
	}

	return nil, errors.New("comment not found")
}

// ParseForm populates a Comment from a form.
func (c *Comment) ParseForm(r *http.Request) {
	// Helper function to set an attribute only if the
	// attribute is currently empty.
	define := func(target *string, value string) {
		if value != "" {
			log.Info("SET DEFINE: %s", value)
			*target = value
		}
	}

	define(&c.ThreadID, r.FormValue("thread"))
	define(&c.OriginURL, r.FormValue("origin"))
	define(&c.Subject, r.FormValue("subject"))

	define(&c.Name, r.FormValue("name"))
	define(&c.Email, r.FormValue("email"))
	define(&c.Body, r.FormValue("body"))
	c.Subscribe = r.FormValue("subscribe") == "true"

	// When editing a post
	c.Editing = r.FormValue("editing") == "true"

	c.Trap1 = r.FormValue("url")
	c.Trap2 = r.FormValue("comment")

	// Default the timestamp values.
	if c.Created.IsZero() {
		c.Created = time.Now().UTC()
		c.Updated = c.Created
	} else {
		c.Updated = time.Now().UTC()
	}

	c.LoadAvatar()
}

// LoadAvatar calculates the user's avatar for the comment.
func (c *Comment) LoadAvatar() {
	// MD5 hash the email address for Gravatar.
	if _, err := mail.ParseAddress(c.Email); err == nil {
		h := md5.New()
		io.WriteString(h, c.Email)
		hash := fmt.Sprintf("%x", h.Sum(nil))
		c.Avatar = fmt.Sprintf(
			"//www.gravatar.com/avatar/%s?s=96",
			hash,
		)
	} else {
		// Default gravatar.
		c.Avatar = "https://www.gravatar.com/avatar/00000000000000000000000000000000"
	}
}

// Validate checks the comment's fields for validity.
func (c *Comment) Validate() error {
	// Spambot trap fields.
	if c.Trap1 != "http://" || c.Trap2 != "" {
		return errors.New("find a human")
	}

	// Required metadata fields.
	if len(c.ThreadID) == 0 {
		return errors.New("you lost the comment thread ID")
	} else if len(c.Subject) == 0 {
		return errors.New("this comment thread is missing a subject")
	}

	if len(c.Body) == 0 {
		return errors.New("the message is required")
	}

	return nil
}
