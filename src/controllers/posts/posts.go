package postctl

import (
	"bytes"
	"fmt"
	"html/template"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"github.com/kirsle/blog/src/log"
	"github.com/kirsle/blog/src/markdown"
	"github.com/kirsle/blog/src/middleware/auth"
	"github.com/kirsle/blog/models/posts"
	"github.com/kirsle/blog/models/users"
	"github.com/kirsle/blog/src/render"
	"github.com/kirsle/blog/src/responses"
	"github.com/kirsle/blog/src/types"
	"github.com/urfave/negroni"
)

// PostMeta associates a Post with injected metadata.
type PostMeta struct {
	Post        *posts.Post
	Rendered    template.HTML
	Author      *users.User
	NumComments int
	IndexView   bool
	Snipped     bool
}

// Archive holds data for a piece of the blog archive.
type Archive struct {
	Label string
	Date  time.Time
	Posts []posts.Post
}

// Register the blog routes to the app.
func Register(r *mux.Router, loginError http.HandlerFunc) {
	render.Funcs["RenderIndex"] = partialIndex
	render.Funcs["RenderPost"] = partialPost
	render.Funcs["RenderTags"] = partialTags

	// Public routes
	r.HandleFunc("/blog", indexHandler)
	r.HandleFunc("/blog.rss", feedHandler)
	r.HandleFunc("/blog.atom", feedHandler)
	r.HandleFunc("/archive", archiveHandler)
	r.HandleFunc("/tagged", taggedHandler)
	r.HandleFunc("/tagged/{tag}", taggedHandler)
	r.HandleFunc("/blog/category/{tag}", func(w http.ResponseWriter, r *http.Request) {
		params := mux.Vars(r)
		tag, ok := params["tag"]
		if !ok {
			responses.NotFound(w, r, "Not Found")
			return
		}
		responses.Redirect(w, "/tagged/"+tag)
	})
	r.HandleFunc("/blog/entry/{fragment}", func(w http.ResponseWriter, r *http.Request) {
		params := mux.Vars(r)
		fragment, ok := params["fragment"]
		if !ok {
			responses.NotFound(w, r, "Not Found")
			return
		}
		responses.Redirect(w, "/"+fragment)
	})

	// Login-required routers.
	loginRouter := mux.NewRouter()
	loginRouter.HandleFunc("/blog/edit", editHandler)
	loginRouter.HandleFunc("/blog/delete", deleteHandler)
	loginRouter.HandleFunc("/blog/drafts", drafts)
	loginRouter.HandleFunc("/blog/private", privatePosts)
	r.PathPrefix("/blog").Handler(
		negroni.New(
			negroni.HandlerFunc(auth.LoginRequired(loginError)),
			negroni.Wrap(loginRouter),
		),
	)
}

// RecentPosts gets and filters the blog entries and orders them by most recent.
func RecentPosts(r *http.Request, tag, privacy string) []posts.Post {
	// Get the blog index.
	idx, _ := posts.GetIndex()

	// The set of blog posts to show.
	var pool []posts.Post
	for _, post := range idx.Posts {
		// Limiting by a specific privacy setting? (drafts or private only)
		if privacy != "" {
			switch privacy {
			case types.DRAFT:
				if post.Privacy != types.DRAFT {
					continue
				}
			case types.PRIVATE:
				if post.Privacy != types.PRIVATE && post.Privacy != types.UNLISTED {
					continue
				}
			}
		} else {
			// Exclude certain posts in generic index views.
			if (post.Privacy == types.PRIVATE || post.Privacy == types.UNLISTED) && !auth.LoggedIn(r) {
				continue
			} else if post.Privacy == types.DRAFT {
				continue
			}
		}

		// Limit by tag?
		if tag != "" {
			var tagMatch bool
			if tag != "" {
				for _, check := range post.Tags {
					if check == tag {
						tagMatch = true
						break
					}
				}
			}

			if !tagMatch {
				continue
			}
		}

		pool = append(pool, post)
	}

	sort.Sort(sort.Reverse(posts.ByUpdated(pool)))
	return pool
}

// ViewPost is the underlying implementation of the handler to view a blog
// post, so that it can be called from non-http.HandlerFunc contexts.
// Specifically, from the catch-all page handler to allow blog URL fragments
// to map to their post.
func ViewPost(w http.ResponseWriter, r *http.Request, fragment string) error {
	post, err := posts.LoadFragment(fragment)
	if err != nil {
		return err
	}

	// Handle post privacy.
	if post.Privacy == types.PRIVATE || post.Privacy == types.DRAFT {
		if !auth.LoggedIn(r) {
			responses.NotFound(w, r, "That post is not public.")
			return nil
		}
	}

	v := map[string]interface{}{
		"Post": post,
	}
	render.Template(w, r, "blog/entry", v)

	return nil
}

// partialPost renders a blog post as a partial template and returns the HTML.
// If indexView is true, the blog headers will be hyperlinked to the dedicated
// entry view page.
func partialPost(r *http.Request, p *posts.Post, indexView bool, numComments int) template.HTML {
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
			parts := strings.SplitN(p.Body, "<snip>", 2)
			p.Body = parts[0]
			snipped = true
		}
	}

	p.Body = strings.Replace(p.Body, "<snip>", "<div id=\"snip\"></div>", 1)

	// Render the post to HTML.
	var rendered template.HTML
	if p.ContentType == string(types.MARKDOWN) {
		rendered = template.HTML(markdown.RenderTrustedMarkdown(p.Body))
	} else {
		rendered = template.HTML(p.Body)
	}

	meta := map[string]interface{}{
		"Post":        p,
		"Rendered":    rendered,
		"Author":      author,
		"IndexView":   indexView,
		"Snipped":     snipped,
		"NumComments": numComments,
	}
	output := bytes.Buffer{}
	err = render.Template(&output, r, "blog/entry.partial", meta)
	if err != nil {
		return template.HTML(fmt.Sprintf("[template error in blog/entry.partial: %s]", err.Error()))
	}

	return template.HTML(output.String())
}
