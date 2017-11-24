package core

import (
	"bytes"
	"errors"
	"html/template"
	"net/http"
	"sort"
	"strconv"
	"strings"

	"github.com/gorilla/mux"
	"github.com/kirsle/blog/core/models/posts"
	"github.com/kirsle/blog/core/models/users"
	"github.com/urfave/negroni"
)

// PostMeta associates a Post with injected metadata.
type PostMeta struct {
	Post      *posts.Post
	Rendered  template.HTML
	Author    *users.User
	IndexView bool
	Snipped   bool
}

// BlogRoutes attaches the blog routes to the app.
func (b *Blog) BlogRoutes(r *mux.Router) {
	// Public routes
	r.HandleFunc("/blog", b.BlogIndex)

	// Login-required routers.
	loginRouter := mux.NewRouter()
	loginRouter.HandleFunc("/blog/edit", b.EditBlog)
	loginRouter.HandleFunc("/blog/delete", b.DeletePost)
	r.PathPrefix("/blog").Handler(
		negroni.New(
			negroni.HandlerFunc(b.LoginRequired),
			negroni.Wrap(loginRouter),
		),
	)

	adminRouter := mux.NewRouter().PathPrefix("/admin").Subrouter().StrictSlash(false)
	r.HandleFunc("/admin", b.AdminHandler) // so as to not be "/admin/"
	adminRouter.HandleFunc("/settings", b.SettingsHandler)
	adminRouter.PathPrefix("/").HandlerFunc(b.PageHandler)
	r.PathPrefix("/admin").Handler(negroni.New(
		negroni.HandlerFunc(b.LoginRequired),
		negroni.Wrap(adminRouter),
	))
}

// BlogIndex renders the main index page of the blog.
func (b *Blog) BlogIndex(w http.ResponseWriter, r *http.Request) {
	v := NewVars(map[interface{}]interface{}{})

	// Get the blog index.
	idx, _ := posts.GetIndex()

	// The set of blog posts to show.
	var pool []posts.Post
	for _, post := range idx.Posts {
		pool = append(pool, post)
	}

	sort.Sort(sort.Reverse(posts.ByUpdated(pool)))

	// Query parameters.
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page <= 0 {
		page = 1
	}
	perPage := 5 // TODO: configurable
	offset := (page - 1) * perPage
	stop := offset + perPage

	// Handle pagination.
	v.Data["Page"] = page
	if page > 1 {
		v.Data["PreviousPage"] = page - 1
	} else {
		v.Data["PreviousPage"] = 0
	}
	if offset+perPage < len(pool) {
		v.Data["NextPage"] = page + 1
	} else {
		v.Data["NextPage"] = 0
	}

	var view []PostMeta
	for i := offset; i < stop; i++ {
		if i >= len(pool) {
			continue
		}
		post, err := posts.Load(pool[i].ID)
		if err != nil {
			log.Error("couldn't load full post data for ID %d (found in index.json)", pool[i].ID)
			continue
		}
		var rendered template.HTML

		// Body has a snipped section?
		if strings.Contains(post.Body, "<snip>") {
			parts := strings.SplitN(post.Body, "<snip>", 1)
			post.Body = parts[0]
		}

		// Render the post.
		if post.ContentType == "markdown" {
			rendered = template.HTML(b.RenderTrustedMarkdown(post.Body))
		} else {
			rendered = template.HTML(post.Body)
		}

		// Look up the author's information.
		author, err := users.LoadReadonly(post.AuthorID)
		if err != nil {
			log.Error("Failed to look up post author ID %d (post %d): %v", post.AuthorID, post.ID, err)
			author = users.DeletedUser()
		}

		view = append(view, PostMeta{
			Post:     post,
			Rendered: rendered,
			Author:   author,
		})
	}

	v.Data["View"] = view
	b.RenderTemplate(w, r, "blog/index", v)
}

// viewPost is the underlying implementation of the handler to view a blog
// post, so that it can be called from non-http.HandlerFunc contexts.
func (b *Blog) viewPost(w http.ResponseWriter, r *http.Request, fragment string) error {
	post, err := posts.LoadFragment(fragment)
	if err != nil {
		return err
	}

	v := NewVars(map[interface{}]interface{}{
		"Post": post,
	})
	b.RenderTemplate(w, r, "blog/entry", v)

	return nil
}

// RenderPost renders a blog post as a partial template and returns the HTML.
// If indexView is true, the blog headers will be hyperlinked to the dedicated
// entry view page.
func (b *Blog) RenderPost(p *posts.Post, indexView bool) template.HTML {
	// Look up the author's information.
	author, err := users.LoadReadonly(p.AuthorID)
	if err != nil {
		log.Error("Failed to look up post author ID %d (post %d): %v", p.AuthorID, p.ID, err)
		author = users.DeletedUser()
	}

	// "Read More" snippet for index views.
	var snipped bool
	if indexView {
		if strings.Contains(p.Body, "<snip>") {
			log.Warn("HAS SNIP TAG!")
			parts := strings.SplitN(p.Body, "<snip>", 2)
			p.Body = parts[0]
			snipped = true
		}
	}

	// Render the post to HTML.
	var rendered template.HTML
	if p.ContentType == "markdown" {
		rendered = template.HTML(b.RenderTrustedMarkdown(p.Body))
	} else {
		rendered = template.HTML(p.Body)
	}

	// Get the template snippet.
	filepath, err := b.ResolvePath("blog/entry.partial")
	if err != nil {
		log.Error(err.Error())
		return "[error: missing blog/entry.partial]"
	}
	t := template.New("entry.partial.gohtml")
	t, err = t.ParseFiles(filepath.Absolute)
	if err != nil {
		log.Error("Failed to parse entry.partial: %s", err.Error())
		return "[error parsing template in blog/entry.partial]"
	}

	meta := PostMeta{
		Post:      p,
		Rendered:  rendered,
		Author:    author,
		IndexView: indexView,
		Snipped:   snipped,
	}
	output := bytes.Buffer{}
	err = t.Execute(&output, meta)
	if err != nil {
		log.Error(err.Error())
		return "[error executing template in blog/entry.partial]"
	}

	return template.HTML(output.String())
}

// EditBlog is the blog writing and editing page.
func (b *Blog) EditBlog(w http.ResponseWriter, r *http.Request) {
	v := NewVars(map[interface{}]interface{}{
		"preview": "",
	})
	var post *posts.Post

	// Are we editing an existing post?
	if idStr := r.URL.Query().Get("id"); idStr != "" {
		id, err := strconv.Atoi(idStr)
		if err == nil {
			post, err = posts.Load(id)
			if err != nil {
				v.Error = errors.New("that post ID was not found")
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
			if post.ContentType == "markdown" || post.ContentType == "markdown+html" {
				v.Data["preview"] = template.HTML(b.RenderMarkdown(post.Body))
			} else {
				v.Data["preview"] = template.HTML(post.Body)
			}
		case "post":
			if err := post.Validate(); err != nil {
				v.Error = err
			} else {
				author, _ := b.CurrentUser(r)
				post.AuthorID = author.ID
				err = post.Save()
				if err != nil {
					v.Error = err
				} else {
					b.Flash(w, r, "Post created!")
					b.Redirect(w, "/"+post.Fragment)
				}
			}
		}
	}

	v.Data["post"] = post
	b.RenderTemplate(w, r, "blog/edit", v)
}

// DeletePost to delete a blog entry.
func (b *Blog) DeletePost(w http.ResponseWriter, r *http.Request) {
	var post *posts.Post
	v := NewVars(map[interface{}]interface{}{
		"Post": nil,
	})

	var idStr string
	if r.Method == http.MethodPost {
		idStr = r.FormValue("id")
	} else {
		idStr = r.URL.Query().Get("id")
	}
	if idStr == "" {
		b.FlashAndRedirect(w, r, "/admin", "No post ID given for deletion!")
		return
	}

	// Convert the post ID to an int.
	id, err := strconv.Atoi(idStr)
	if err == nil {
		post, err = posts.Load(id)
		if err != nil {
			b.FlashAndRedirect(w, r, "/admin", "That post ID was not found.")
			return
		}
	}

	if r.Method == http.MethodPost {
		post.Delete()
		b.FlashAndRedirect(w, r, "/admin", "Blog entry deleted!")
		return
	}

	v.Data["Post"] = post
	b.RenderTemplate(w, r, "blog/delete", v)
}
