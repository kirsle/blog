package posts

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

// Regexp used to parse a thumbnail image from a blog post. Looks for the first
// URI component ending with an image extension.
var (
	ThumbnailImageRegexp = regexp.MustCompile(`['"(]([a-zA-Z0-9-_:/?.=&]+\.(?:jpe?g|png|gif))['")]`)
)

func init() {
	log = golog.GetLogger("blog")
}

// Post holds information for a blog post.
type Post struct {
	ID             int       `json:"id"`
	Title          string    `json:"title"`
	Fragment       string    `json:"fragment"`
	ContentType    string    `json:"contentType,omitempty"`
	AuthorID       int       `json:"author"`
	Body           string    `json:"body,omitempty"`
	Privacy        string    `json:"privacy"`
	Sticky         bool      `json:"sticky"`
	EnableComments bool      `json:"enableComments"`
	Tags           []string  `json:"tags"`
	Created        time.Time `json:"created"`
	Updated        time.Time `json:"updated"`
}

// New creates a blank post with sensible defaults.
func New() *Post {
	return &Post{
		ContentType:    "markdown",
		Privacy:        "public",
		EnableComments: true,
	}
}

// ParseForm populates the post from form values.
func (p *Post) ParseForm(r *http.Request) {
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
	if p.Privacy != "public" && p.Privacy != "draft" && p.Privacy != "private" && p.Privacy != "unlisted" {
		return errors.New("invalid setting for Privacy")
	}
	return nil
}

// Load a post by its ID.
func Load(id int) (*Post, error) {
	p := &Post{}
	err := DB.Get(fmt.Sprintf("blog/posts/%d", id), &p)
	return p, err
}

// LoadFragment loads a blog entry by its URL fragment.
func LoadFragment(fragment string) (*Post, error) {
	idx, err := GetIndex()
	if err != nil {
		return nil, err
	}

	if postID, ok := idx.Fragments[fragment]; ok {
		return Load(postID)
	}

	return nil, errors.New("no such fragment found")
}

// Save the blog post.
func (p *Post) Save() error {
	// Editing an existing post?
	if p.ID == 0 {
		p.ID = p.nextID()
	}

	// Generate a URL fragment if needed.
	if p.Fragment == "" {
		fragment := strings.ToLower(p.Title)
		fragment = regexp.MustCompile(`[^A-Za-z0-9]+`).ReplaceAllString(fragment, "-")
		if strings.Contains(fragment, "--") {
			log.Error("Generated blog fragment '%s' contains double dashes still!", fragment)
		}
		p.Fragment = strings.Trim(fragment, "-")

		// If still no fragment, make one based on the post ID.
		if p.Fragment == "" {
			p.Fragment = fmt.Sprintf("post-%d", p.ID)
		}
	}

	// Make sure the URL fragment is unique!
	if len(p.Fragment) > 0 {
		if exist, err := LoadFragment(p.Fragment); err == nil && exist.ID != p.ID {
			var resolved bool
			for i := 1; i <= 100; i++ {
				fragment := fmt.Sprintf("%s-%d", p.Fragment, i)
				_, err := LoadFragment(fragment)
				if err == nil {
					continue
				}

				p.Fragment = fragment
				resolved = true
				break
			}

			if !resolved {
				return fmt.Errorf("failed to generate a unique URL fragment for '%s' after 100 attempts", p.Fragment)
			}
		}
	}

	// Dates & times.
	if p.Created.IsZero() {
		p.Created = time.Now().UTC()
	}
	if p.Updated.IsZero() {
		p.Updated = p.Created
	}

	// Empty tag lists.
	if len(p.Tags) == 1 && p.Tags[0] == "" {
		p.Tags = []string{}
	}

	// Write the post.
	DB.Commit(fmt.Sprintf("blog/posts/%d", p.ID), p)

	// Update the index cache.
	err := UpdateIndex(p)
	if err != nil {
		return fmt.Errorf("RebuildIndex() error: %v", err)
	}

	return nil
}

// Delete a blog entry.
func (p *Post) Delete() error {
	if p.ID == 0 {
		return errors.New("post has no ID")
	}

	// Delete the DB files.
	DB.Delete(fmt.Sprintf("blog/posts/%d", p.ID))
	DB.Delete(fmt.Sprintf("blog/fragments/%s", p.Fragment))

	// Remove it from the index.
	idx, err := GetIndex()
	if err != nil {
		return fmt.Errorf("GetIndex error: %v", err)
	}
	return idx.Delete(p)
}

// ExtractThumbnail searches and returns a thumbnail image to represent the
// post. This will be the first image embedded in the post, or nothing.
func (p *Post) ExtractThumbnail() (string, bool) {
	result := ThumbnailImageRegexp.FindStringSubmatch(p.Body)
	if len(result) < 2 {
		return "", false
	}
	return result[1], true
}

// getNextID gets the next blog post ID.
func (p *Post) nextID() int {
	// Highest ID seen so far.
	var highest int

	posts, err := DB.List("blog/posts")
	if err != nil {
		return 1
	}

	for _, doc := range posts {
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
