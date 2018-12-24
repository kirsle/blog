package comments

import (
	"bytes"
	"html/template"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/kirsle/blog/src/log"
	"github.com/kirsle/blog/src/mail"
	"github.com/kirsle/blog/src/markdown"
	"github.com/kirsle/blog/src/middleware/auth"
	"github.com/kirsle/blog/models/comments"
	"github.com/kirsle/blog/models/users"
	"github.com/kirsle/blog/src/render"
	"github.com/kirsle/blog/src/responses"
	"github.com/kirsle/blog/src/sessions"
)

var badRequest func(http.ResponseWriter, *http.Request, string)

// Register the comment routes to the app.
func Register(r *mux.Router) {
	badRequest = responses.BadRequest
	render.Funcs["RenderComments"] = RenderComments

	r.HandleFunc("/comments", commentHandler)
	r.HandleFunc("/comments/subscription", subscriptionHandler)
	r.HandleFunc("/comments/quick-delete", quickDeleteHandler)
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
func RenderComments(r *http.Request, subject string, ids ...string) template.HTML {
	id := strings.Join(ids, "-")
	session := sessions.Get(r)
	url := r.URL.Path

	// Load their cached name and email if they posted a comment before.
	name, _ := session.Values["c.name"].(string)
	email, _ := session.Values["c.email"].(string)
	editToken, _ := session.Values["c.token"].(string)
	csrf, _ := session.Values["csrf"].(string)

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
		c.HTML = template.HTML(markdown.RenderMarkdown(c.Body))
		c.ThreadID = thread.ID
		c.OriginURL = url
		c.CSRF = csrf

		// Look up the author username.
		if c.UserID > 0 {
			if _, ok := userMap[c.UserID]; !ok {
				if user, err2 := users.Load(c.UserID); err2 == nil {
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
	filepath, err := render.ResolvePath("comments/comments.partial")
	if err != nil {
		log.Error(err.Error())
		return template.HTML("[error: missing comments/comments.partial]")
	}

	// And the comment view partial.
	entryPartial, err := render.ResolvePath("comments/entry.partial")
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
		CSRF:      csrf,
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

func commentHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		badRequest(w, r, "That method is not allowed.")
		return
	}
	currentUser, _ := auth.CurrentUser(r)
	editToken := getEditToken(w, r)
	submit := r.FormValue("submit")

	// Load the comment data from the form.
	c := &comments.Comment{}
	c.ParseForm(r)
	if c.ThreadID == "" {
		responses.FlashAndRedirect(w, r, "/", "No thread ID found in the comment form.")
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
			responses.FlashAndRedirect(w, r, "/", "That comment was not found.")
			return
		}

		// Verify they have the matching edit token. Admin users are allowed.
		if c.EditToken != editToken && !currentUser.Admin {
			responses.FlashAndRedirect(w, r, origin, "You don't have permission to edit that comment.")
			return
		}

		// Parse the extra form data into the comment struct.
		c.ParseForm(r)
	}

	// Are we deleting said post?
	if submit == "confirm-delete" {
		t.Delete(c.ID)
		responses.FlashAndRedirect(w, r, origin, "Comment deleted!")
		return
	}

	// Cache their name and email in their session.
	session := sessions.Get(r)
	session.Values["c.name"] = c.Name
	session.Values["c.email"] = c.Email
	session.Save(r, w)

	v := map[string]interface{}{}

	// Previewing, deleting, or posting?
	switch submit {
	case "preview", "delete":
		if !c.Editing && currentUser.IsAuthenticated {
			c.Name = currentUser.Name
			c.Email = currentUser.Email
			c.LoadAvatar()
		}
		c.HTML = template.HTML(markdown.RenderMarkdown(c.Body))
	case "post":
		if err := c.Validate(); err != nil {
			v["Error"] = err
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
			err := t.Post(c)
			if err != nil {
				responses.FlashAndRedirect(w, r, c.OriginURL, "Error posting comment: %s", err)
				return
			}
			mail.NotifyComment(c)

			// Are they subscribing to future comments?
			if c.Subscribe && len(c.Email) > 0 {
				if _, err := mail.ParseAddress(c.Email); err == nil {
					m := comments.LoadMailingList()
					m.Subscribe(t.ID, c.Email)
					responses.FlashAndRedirect(w, r, c.OriginURL,
						"Comment posted, and you've been subscribed to "+
							"future comments on this page.",
					)
					return
				}
			}
			responses.FlashAndRedirect(w, r, c.OriginURL, "Comment posted!")
			log.Info("t: %v", t.Comments)
			return
		}
	}

	v["Thread"] = t
	v["Comment"] = c
	v["Editing"] = c.Editing
	v["Deleting"] = submit == "delete"

	render.Template(w, r, "comments/index.gohtml", v)
}

func quickDeleteHandler(w http.ResponseWriter, r *http.Request) {
	thread := r.URL.Query().Get("t")
	token := r.URL.Query().Get("d")
	if thread == "" || token == "" {
		badRequest(w, r, "Bad Request")
		return
	}

	t, err := comments.Load(thread)
	if err != nil {
		badRequest(w, r, "Comment thread does not exist.")
		return
	}

	if c, err := t.FindByDeleteToken(token); err == nil {
		t.Delete(c.ID)
	}

	responses.FlashAndRedirect(w, r, "/", "Comment deleted!")
}

// getEditToken gets or generates an edit token from the user's session, which
// allows a user to edit their comment for a short while after they post it.
func getEditToken(w http.ResponseWriter, r *http.Request) string {
	session := sessions.Get(r)
	if token, ok := session.Values["c.token"].(string); ok && len(token) > 0 {
		return token
	}

	token := uuid.New().String()
	session.Values["c.token"] = token
	session.Save(r, w)
	return token
}
