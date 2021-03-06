package postctl

import (
	"errors"
	"html/template"
	"net/http"
	"strconv"
	"time"

	"github.com/kirsle/blog/src/markdown"
	"github.com/kirsle/blog/src/middleware/auth"
	"github.com/kirsle/blog/src/render"
	"github.com/kirsle/blog/src/responses"
	"github.com/kirsle/blog/src/types"
	"github.com/kirsle/blog/models/posts"
)

// editHandler is the blog writing and editing page.
func editHandler(w http.ResponseWriter, r *http.Request) {
	v := map[string]interface{}{
		"preview": "",
	}
	var post *posts.Post
	var isNew bool

	// Are we editing an existing post?
	if idStr := r.FormValue("id"); idStr != "" {
		id, err := strconv.Atoi(idStr)
		if err == nil {
			post, err = posts.Load(id)
			if err != nil {
				v["Error"] = errors.New("that post ID was not found")
				post = posts.New()
				isNew = true
			}
		}
	} else {
		post = posts.New()
		isNew = true
	}

	if r.Method == http.MethodPost {
		// Parse from form values.
		post.ParseForm(r)

		// Previewing, or submitting?
		switch r.FormValue("submit") {
		case "preview":
			if post.ContentType == string(types.MARKDOWN) {
				v["preview"] = template.HTML(markdown.RenderTrustedMarkdown(post.Body))
			} else {
				v["preview"] = template.HTML(post.Body)
			}
		case "post":
			if err := post.Validate(); err != nil {
				v["Error"] = err
			} else {
				author, _ := auth.CurrentUser(r)
				post.AuthorID = author.ID

				// When editing, allow to not touch the last updated time.
				if !isNew && r.FormValue("no-update") == "true" {
					post.Updated = post.Created
				} else {
					post.Updated = time.Now().UTC()
				}
				err = post.Save()

				if err != nil {
					v["Error"] = err
				} else {
					responses.Flash(w, r, "Post created!")
					responses.Redirect(w, "/"+post.Fragment)
				}
			}
		}
	}

	v["post"] = post
	render.Template(w, r, "blog/edit", v)
}

// deleteHandler to delete a blog entry.
func deleteHandler(w http.ResponseWriter, r *http.Request) {
	var post *posts.Post
	v := map[string]interface{}{
		"Post": nil,
	}

	var idStr string
	if r.Method == http.MethodPost {
		idStr = r.FormValue("id")
	} else {
		idStr = r.URL.Query().Get("id")
	}
	if idStr == "" {
		responses.FlashAndRedirect(w, r, "/admin", "No post ID given for deletion!")
		return
	}

	// Convert the post ID to an int.
	id, err := strconv.Atoi(idStr)
	if err == nil {
		post, err = posts.Load(id)
		if err != nil {
			responses.FlashAndRedirect(w, r, "/admin", "That post ID was not found.")
			return
		}
	}

	if r.Method == http.MethodPost {
		post.Delete()
		responses.FlashAndRedirect(w, r, "/admin", "Blog entry deleted!")
		return
	}

	v["Post"] = post
	render.Template(w, r, "blog/delete", v)
}
