package postctl

import (
	"errors"
	"html/template"
	"net/http"
	"strconv"
	"time"

	"github.com/kirsle/blog/internal/markdown"
	"github.com/kirsle/blog/internal/middleware/auth"
	"github.com/kirsle/blog/models/posts"
	"github.com/kirsle/blog/internal/render"
	"github.com/kirsle/blog/internal/responses"
	"github.com/kirsle/blog/internal/types"
)

// editHandler is the blog writing and editing page.
func editHandler(w http.ResponseWriter, r *http.Request) {
	v := map[string]interface{}{
		"preview": "",
	}
	var post *posts.Post

	// Are we editing an existing post?
	if idStr := r.FormValue("id"); idStr != "" {
		id, err := strconv.Atoi(idStr)
		if err == nil {
			post, err = posts.Load(id)
			if err != nil {
				v["Error"] = errors.New("that post ID was not found")
				post = posts.New()
			}
		}
	} else {
		post = posts.New()
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

				post.Updated = time.Now().UTC()
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
