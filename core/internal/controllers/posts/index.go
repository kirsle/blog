package postctl

import (
	"bytes"
	"fmt"
	"html/template"
	"net/http"
	"strconv"

	"github.com/kirsle/blog/core/internal/log"
	"github.com/kirsle/blog/core/internal/models/comments"
	"github.com/kirsle/blog/core/internal/models/posts"
	"github.com/kirsle/blog/core/internal/models/users"
	"github.com/kirsle/blog/core/internal/render"
	"github.com/kirsle/blog/core/internal/types"
)

// partialIndex renders and returns the blog index partial.
func partialIndex(r *http.Request, tag, privacy string) template.HTML {
	// Get the recent blog entries, filtered by the tag/privacy settings.
	pool := RecentPosts(r, tag, privacy)
	if len(pool) == 0 {
		return template.HTML("No blog posts were found.")
	}

	// Query parameters.
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page <= 0 {
		page = 1
	}
	perPage := 5 // TODO: configurable
	offset := (page - 1) * perPage
	stop := offset + perPage

	// Handle pagination.
	var previousPage, nextPage int
	if page > 1 {
		previousPage = page - 1
	} else {
		previousPage = 0
	}
	if offset+perPage < len(pool) {
		nextPage = page + 1
	} else {
		nextPage = 0
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

		// Look up the author's information.
		author, err := users.LoadReadonly(post.AuthorID)
		if err != nil {
			log.Error("Failed to look up post author ID %d (post %d): %v", post.AuthorID, post.ID, err)
			author = users.DeletedUser()
		}

		// Count the comments on this post.
		var numComments int
		if thread, err := comments.Load(fmt.Sprintf("post-%d", post.ID)); err == nil {
			numComments = len(thread.Comments)
		}

		view = append(view, PostMeta{
			Post:        post,
			Author:      author,
			NumComments: numComments,
		})
	}

	// Render the blog index partial.
	var output bytes.Buffer
	v := map[string]interface{}{
		"PreviousPage": previousPage,
		"NextPage":     nextPage,
		"View":         view,
	}
	render.Template(&output, r, "blog/index.partial", v)

	return template.HTML(output.String())
}

// indexHandler renders the main index page of the blog.
func indexHandler(w http.ResponseWriter, r *http.Request) {
	commonIndexHandler(w, r, "", "")
}

// drafts renders an index view of only draft posts. Login required.
func drafts(w http.ResponseWriter, r *http.Request) {
	commonIndexHandler(w, r, "", types.DRAFT)
}

// privatePosts renders an index view of only private posts. Login required.
func privatePosts(w http.ResponseWriter, r *http.Request) {
	commonIndexHandler(w, r, "", types.PRIVATE)
}

// commonIndexHandler handles common logic for blog index views.
func commonIndexHandler(w http.ResponseWriter, r *http.Request, tag, privacy string) {
	// Page title.
	var title string
	if privacy == types.DRAFT {
		title = "Draft Posts"
	} else if privacy == types.PRIVATE {
		title = "Private Posts"
	} else if tag != "" {
		title = "Tagged as: " + tag
	} else {
		title = "Blog"
	}

	render.Template(w, r, "blog/index", map[string]interface{}{
		"Title":   title,
		"Tag":     tag,
		"Privacy": privacy,
	})
}
