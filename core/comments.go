package core

import (
	"bytes"
	"errors"
	"html/template"
	"net/http"
	"net/mail"
	"strings"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/gorilla/sessions"
	"github.com/kirsle/blog/core/models/comments"
	"github.com/kirsle/blog/core/models/users"
)

// CommentRoutes attaches the comment routes to the app.
func (b *Blog) CommentRoutes(r *mux.Router) {
	r.HandleFunc("/comments", b.CommentHandler)
	r.HandleFunc("/comments/subscription", b.SubscriptionHandler)
	r.HandleFunc("/comments/quick-delete", b.QuickDeleteHandler)
}

// CommentMeta is the template variables for comment threads.
type CommentMeta struct {
	NewComment comments.Comment
	ID         string
	OriginURL  string // URL where original comment thread appeared
	Subject    string // email subject
	Thread     *comments.Thread
	Authors    map[int]*users.User
	CSRF       string
}

// RenderComments renders a comment form partial and returns the HTML.
func (b *Blog) RenderComments(session *sessions.Session, csrfToken, url, subject string, ids ...string) template.HTML {
	id := strings.Join(ids, "-")

	// Load their cached name and email if they posted a comment before.
	name, _ := session.Values["c.name"].(string)
	email, _ := session.Values["c.email"].(string)
	editToken, _ := session.Values["c.token"].(string)

	// Check if the user is a logged-in admin, to make all comments editable.
	var isAdmin bool
	var isAuthenticated bool
	if loggedIn, ok := session.Values["logged-in"].(bool); ok && loggedIn {
		isAuthenticated = true
		if userID, ok := session.Values["user-id"].(int); ok {
			if user, err := users.Load(userID); err == nil {
				isAdmin = user.Admin
			}
		}
	}

	thread, err := comments.Load(id)
	if err != nil {
		thread = comments.New(id)
	}

	// Render all the comments in the thread.
	userMap := map[int]*users.User{}
	for _, c := range thread.Comments {
		c.HTML = template.HTML(b.RenderMarkdown(c.Body))
		c.ThreadID = thread.ID
		c.OriginURL = url
		c.CSRF = csrfToken

		// Look up the author username.
		if c.UserID > 0 {
			if _, ok := userMap[c.UserID]; !ok {
				if user, err := users.Load(c.UserID); err == nil {
					userMap[c.UserID] = user
				}
			}

			if user, ok := userMap[c.UserID]; ok {
				c.Name = user.Name
				c.Username = user.Username
				c.Email = user.Email
				c.LoadAvatar()
			}
		}

		// Is it editable?
		if isAdmin || (len(c.EditToken) > 0 && c.EditToken == editToken) {
			c.Editable = true
		}
	}

	// Get the template snippet.
	filepath, err := b.ResolvePath("comments/comments.partial")
	if err != nil {
		log.Error(err.Error())
		return template.HTML("[error: missing comments/comments.partial]")
	}

	// And the comment view partial.
	entryPartial, err := b.ResolvePath("comments/entry.partial")
	if err != nil {
		log.Error(err.Error())
		return template.HTML("[error: missing comments/entry.partial]")
	}

	t := template.New("comments.partial.gohtml")
	t, err = t.ParseFiles(entryPartial.Absolute, filepath.Absolute)
	if err != nil {
		log.Error("Failed to parse comments.partial: %s", err.Error())
		return template.HTML("[error parsing template in comments/comments.partial]")
	}

	v := CommentMeta{
		ID:        thread.ID,
		OriginURL: url,
		Subject:   subject,
		CSRF:      csrfToken,
		Thread:    &thread,
		NewComment: comments.Comment{
			Name:            name,
			Email:           email,
			IsAuthenticated: isAuthenticated,
		},
	}

	output := bytes.Buffer{}
	err = t.Execute(&output, v)
	if err != nil {
		return template.HTML(err.Error())
	}

	return template.HTML(output.String())
}

// CommentHandler handles the /comments URI for previewing and posting.
func (b *Blog) CommentHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		b.BadRequest(w, r, "That method is not allowed.")
		return
	}
	v := NewVars()
	currentUser, _ := b.CurrentUser(r)
	editToken := b.GetEditToken(w, r)
	submit := r.FormValue("submit")

	// Load the comment data from the form.
	c := &comments.Comment{}
	c.ParseForm(r)
	if c.ThreadID == "" {
		b.FlashAndRedirect(w, r, "/", "No thread ID found in the comment form.")
		return
	}

	// Look up the thread.
	t, err := comments.Load(c.ThreadID)
	if err != nil {
		t = comments.New(c.ThreadID)
	}

	// Origin URL to redirect them to at the end.
	origin := "/"
	if c.OriginURL != "" {
		origin = c.OriginURL
	}

	// Are we editing a post?
	if r.FormValue("editing") == "true" {
		id := r.FormValue("id")
		c, err = t.Find(id)
		if err != nil {
			b.FlashAndRedirect(w, r, "/", "That comment was not found.")
			return
		}

		// Verify they have the matching edit token. Admin users are allowed.
		if c.EditToken != editToken && !currentUser.Admin {
			b.FlashAndRedirect(w, r, origin, "You don't have permission to edit that comment.")
			return
		}

		// Parse the extra form data into the comment struct.
		c.ParseForm(r)
	}

	// Are we deleting said post?
	if submit == "confirm-delete" {
		t.Delete(c.ID)
		b.FlashAndRedirect(w, r, origin, "Comment deleted!")
		return
	}

	// Cache their name and email in their session.
	session := b.Session(r)
	session.Values["c.name"] = c.Name
	session.Values["c.email"] = c.Email
	session.Save(r, w)

	// Previewing, deleting, or posting?
	switch submit {
	case "preview", "delete":
		if !c.Editing && currentUser.IsAuthenticated {
			c.Name = currentUser.Name
			c.Email = currentUser.Email
			c.LoadAvatar()
		}
		c.HTML = template.HTML(b.RenderMarkdown(c.Body))
	case "post":
		if err := c.Validate(); err != nil {
			v.Error = err
		} else {
			// Store our edit token, if we don't have one. For example, admins
			// can edit others' comments but should not replace their edit token.
			if c.EditToken == "" {
				c.EditToken = editToken
			}

			// If we're logged in, tag our user ID with this post.
			if !c.Editing && c.UserID == 0 && currentUser.IsAuthenticated {
				c.UserID = currentUser.ID
			}

			// Append their comment.
			t.Post(c)
			b.NotifyComment(c)

			// Are they subscribing to future comments?
			if c.Subscribe && len(c.Email) > 0 {
				if _, err := mail.ParseAddress(c.Email); err == nil {
					m := comments.LoadMailingList()
					m.Subscribe(t.ID, c.Email)
					b.FlashAndRedirect(w, r, c.OriginURL,
						"Comment posted, and you've been subscribed to "+
							"future comments on this page.",
					)
					return
				}
			}
			b.FlashAndRedirect(w, r, c.OriginURL, "Comment posted!")
			return
		}
	}

	v.Data["Thread"] = t
	v.Data["Comment"] = c
	v.Data["Editing"] = c.Editing
	v.Data["Deleting"] = submit == "delete"

	b.RenderTemplate(w, r, "comments/index.gohtml", v)
}

// SubscriptionHandler to opt out of subscriptions.
func (b *Blog) SubscriptionHandler(w http.ResponseWriter, r *http.Request) {
	v := NewVars()

	// POST to unsubscribe from all threads.
	if r.Method == http.MethodPost {
		email := r.FormValue("email")
		if email == "" {
			v.Error = errors.New("email address is required to unsubscribe from comment threads")
		} else if _, err := mail.ParseAddress(email); err != nil {
			v.Error = errors.New("invalid email address")
		}

		m := comments.LoadMailingList()
		m.UnsubscribeAll(email)
		b.FlashAndRedirect(w, r, "/comments/subscription",
			"You have been unsubscribed from all mailing lists.",
		)
		return
	}

	// GET to unsubscribe from a single thread.
	thread := r.URL.Query().Get("t")
	email := r.URL.Query().Get("e")
	if thread != "" && email != "" {
		m := comments.LoadMailingList()
		m.Unsubscribe(thread, email)
		b.FlashAndRedirect(w, r, "/comments/subscription", "You have been unsubscribed successfully.")
		return
	}

	b.RenderTemplate(w, r, "comments/subscription.gohtml", v)
}

// QuickDeleteHandler allows the admin to quickly delete spam without logging in.
func (b *Blog) QuickDeleteHandler(w http.ResponseWriter, r *http.Request) {
	thread := r.URL.Query().Get("t")
	token := r.URL.Query().Get("d")
	if thread == "" || token == "" {
		b.BadRequest(w, r)
		return
	}

	t, err := comments.Load(thread)
	if err != nil {
		b.BadRequest(w, r, "Comment thread does not exist.")
		return
	}

	if c, err := t.FindByDeleteToken(token); err == nil {
		t.Delete(c.ID)
	}

	b.FlashAndRedirect(w, r, "/", "Comment deleted!")
}

// GetEditToken gets or generates an edit token from the user's session, which
// allows a user to edit their comment for a short while after they post it.
func (b *Blog) GetEditToken(w http.ResponseWriter, r *http.Request) string {
	session := b.Session(r)
	if token, ok := session.Values["c.token"].(string); ok && len(token) > 0 {
		return token
	}

	token := uuid.New().String()
	session.Values["c.token"] = token
	session.Save(r, w)
	return token
}
