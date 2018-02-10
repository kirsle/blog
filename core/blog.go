package core

import (
	"bytes"
	"errors"
	"fmt"
	"html/template"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/feeds"
	"github.com/gorilla/mux"
	"github.com/kirsle/blog/core/models/comments"
	"github.com/kirsle/blog/core/models/posts"
	"github.com/kirsle/blog/core/models/settings"
	"github.com/kirsle/blog/core/models/users"
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

// BlogRoutes attaches the blog routes to the app.
func (b *Blog) BlogRoutes(r *mux.Router) {
	// Public routes
	r.HandleFunc("/blog", b.IndexHandler)
	r.HandleFunc("/blog.rss", b.RSSHandler)
	r.HandleFunc("/blog.atom", b.RSSHandler)
	r.HandleFunc("/archive", b.BlogArchive)
	r.HandleFunc("/tagged", b.Tagged)
	r.HandleFunc("/tagged/{tag}", b.Tagged)
	r.HandleFunc("/blog/category/{tag}", func(w http.ResponseWriter, r *http.Request) {
		params := mux.Vars(r)
		tag, ok := params["tag"]
		if !ok {
			b.NotFound(w, r, "Not Found")
			return
		}
		b.Redirect(w, "/tagged/"+tag)
	})
	r.HandleFunc("/blog/entry/{fragment}", func(w http.ResponseWriter, r *http.Request) {
		params := mux.Vars(r)
		fragment, ok := params["fragment"]
		if !ok {
			b.NotFound(w, r, "Not Found")
			return
		}
		b.Redirect(w, "/"+fragment)
	})

	// Login-required routers.
	loginRouter := mux.NewRouter()
	loginRouter.HandleFunc("/blog/edit", b.EditBlog)
	loginRouter.HandleFunc("/blog/delete", b.DeletePost)
	loginRouter.HandleFunc("/blog/drafts", b.Drafts)
	loginRouter.HandleFunc("/blog/private", b.PrivatePosts)
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

// RSSHandler renders an RSS feed from the blog.
func (b *Blog) RSSHandler(w http.ResponseWriter, r *http.Request) {
	config, _ := settings.Load()
	admin, err := users.Load(1)
	if err != nil {
		b.Error(w, r, "Blog isn't ready yet.")
		return
	}

	feed := &feeds.Feed{
		Title:       config.Site.Title,
		Link:        &feeds.Link{Href: config.Site.URL},
		Description: config.Site.Description,
		Author: &feeds.Author{
			Name:  admin.Name,
			Email: admin.Email,
		},
		Created: time.Now(),
	}

	feed.Items = []*feeds.Item{}
	for i, p := range b.RecentPosts(r, "", "") {
		post, _ := posts.Load(p.ID)
		var suffix string
		if strings.Contains(post.Body, "<snip>") {
			post.Body = strings.Split(post.Body, "<snip>")[0]
			suffix = "..."
		}

		feed.Items = append(feed.Items, &feeds.Item{
			Title:       p.Title,
			Link:        &feeds.Link{Href: config.Site.URL + p.Fragment},
			Description: post.Body + suffix,
			Created:     p.Created,
		})
		if i >= 5 {
			break
		}
	}

	// What format to encode it in?
	if strings.Contains(r.URL.Path, ".atom") {
		atom, _ := feed.ToAtom()
		w.Header().Set("Content-Type", "application/atom+xml")
		w.Write([]byte(atom))
	} else {
		rss, _ := feed.ToRss()
		w.Header().Set("Content-Type", "application/rss+xml")
		w.Write([]byte(rss))
	}
}

// IndexHandler renders the main index page of the blog.
func (b *Blog) IndexHandler(w http.ResponseWriter, r *http.Request) {
	b.CommonIndexHandler(w, r, "", "")
}

// Tagged lets you browse blog posts by category.
func (b *Blog) Tagged(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	tag, ok := params["tag"]
	if !ok {
		// They're listing all the tags.
		b.RenderTemplate(w, r, "blog/tags.gohtml", NewVars())
		return
	}

	b.CommonIndexHandler(w, r, tag, "")
}

// Drafts renders an index view of only draft posts. Login required.
func (b *Blog) Drafts(w http.ResponseWriter, r *http.Request) {
	b.CommonIndexHandler(w, r, "", DRAFT)
}

// PrivatePosts renders an index view of only private posts. Login required.
func (b *Blog) PrivatePosts(w http.ResponseWriter, r *http.Request) {
	b.CommonIndexHandler(w, r, "", PRIVATE)
}

// CommonIndexHandler handles common logic for blog index views.
func (b *Blog) CommonIndexHandler(w http.ResponseWriter, r *http.Request, tag, privacy string) {
	// Page title.
	var title string
	if privacy == DRAFT {
		title = "Draft Posts"
	} else if privacy == PRIVATE {
		title = "Private Posts"
	} else if tag != "" {
		title = "Tagged as: " + tag
	} else {
		title = "Blog"
	}

	b.RenderTemplate(w, r, "blog/index", NewVars(map[interface{}]interface{}{
		"Title":   title,
		"Tag":     tag,
		"Privacy": privacy,
	}))
}

// RecentPosts gets and filters the blog entries and orders them by most recent.
func (b *Blog) RecentPosts(r *http.Request, tag, privacy string) []posts.Post {
	// Get the blog index.
	idx, _ := posts.GetIndex()

	// The set of blog posts to show.
	var pool []posts.Post
	for _, post := range idx.Posts {
		// Limiting by a specific privacy setting? (drafts or private only)
		if privacy != "" {
			switch privacy {
			case DRAFT:
				if post.Privacy != DRAFT {
					continue
				}
			case PRIVATE:
				if post.Privacy != PRIVATE && post.Privacy != UNLISTED {
					continue
				}
			}
		} else {
			// Exclude certain posts in generic index views.
			if (post.Privacy == PRIVATE || post.Privacy == UNLISTED) && !b.LoggedIn(r) {
				continue
			} else if post.Privacy == DRAFT {
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

// RenderIndex renders and returns the blog index partial.
func (b *Blog) RenderIndex(r *http.Request, tag, privacy string) template.HTML {
	// Get the recent blog entries, filtered by the tag/privacy settings.
	pool := b.RecentPosts(r, tag, privacy)
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
	b.RenderPartialTemplate(&output, "blog/index.partial", v, false, nil)

	return template.HTML(output.String())
}

// RenderTags renders the tags partial.
func (b *Blog) RenderTags(r *http.Request, indexView bool) template.HTML {
	idx, err := posts.GetIndex()
	if err != nil {
		return template.HTML("[RenderTags: error getting blog index]")
	}

	tags, err := idx.Tags()
	if err != nil {
		return template.HTML("[RenderTags: error getting tags]")
	}

	var output bytes.Buffer
	v := struct {
		IndexView bool
		Tags      []posts.Tag
	}{
		IndexView: indexView,
		Tags:      tags,
	}
	b.RenderPartialTemplate(&output, "blog/tags.partial", v, false, nil)

	return template.HTML(output.String())
}

// BlogArchive summarizes all blog entries in an archive view.
func (b *Blog) BlogArchive(w http.ResponseWriter, r *http.Request) {
	idx, err := posts.GetIndex()
	if err != nil {
		b.BadRequest(w, r, "Error getting blog index")
		return
	}

	// Group posts by calendar month.
	var months []string
	byMonth := map[string]*Archive{}
	for _, post := range idx.Posts {
		// Exclude certain posts
		if (post.Privacy == PRIVATE || post.Privacy == UNLISTED) && !b.LoggedIn(r) {
			continue
		} else if post.Privacy == DRAFT {
			continue
		}

		label := post.Created.Format("2006-01")
		if _, ok := byMonth[label]; !ok {
			months = append(months, label)
			byMonth[label] = &Archive{
				Label: label,
				Date:  time.Date(post.Created.Year(), post.Created.Month(), post.Created.Day(), 0, 0, 0, 0, time.UTC),
				Posts: []posts.Post{},
			}
		}
		byMonth[label].Posts = append(byMonth[label].Posts, post)
	}

	// Sort the months.
	sort.Sort(sort.Reverse(sort.StringSlice(months)))

	// Prepare the response.
	result := []*Archive{}
	for _, label := range months {
		sort.Sort(sort.Reverse(posts.ByUpdated(byMonth[label].Posts)))
		result = append(result, byMonth[label])
	}

	v := NewVars(map[interface{}]interface{}{
		"Archive": result,
	})
	b.RenderTemplate(w, r, "blog/archive", v)
}

// viewPost is the underlying implementation of the handler to view a blog
// post, so that it can be called from non-http.HandlerFunc contexts.
// Specifically, from the catch-all page handler to allow blog URL fragments
// to map to their post.
func (b *Blog) viewPost(w http.ResponseWriter, r *http.Request, fragment string) error {
	post, err := posts.LoadFragment(fragment)
	if err != nil {
		return err
	}

	// Handle post privacy.
	if post.Privacy == PRIVATE || post.Privacy == DRAFT {
		if !b.LoggedIn(r) {
			b.NotFound(w, r)
			return nil
		}
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
func (b *Blog) RenderPost(p *posts.Post, indexView bool, numComments int) template.HTML {
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
	if p.ContentType == string(MARKDOWN) {
		rendered = template.HTML(b.RenderTrustedMarkdown(p.Body))
	} else {
		rendered = template.HTML(p.Body)
	}

	meta := PostMeta{
		Post:        p,
		Rendered:    rendered,
		Author:      author,
		IndexView:   indexView,
		Snipped:     snipped,
		NumComments: numComments,
	}
	output := bytes.Buffer{}
	err = b.RenderPartialTemplate(&output, "blog/entry.partial", meta, false, nil)
	if err != nil {
		return template.HTML(fmt.Sprintf("[template error in blog/entry.partial: %s]", err.Error()))
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
	if idStr := r.FormValue("id"); idStr != "" {
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
			if post.ContentType == string(MARKDOWN) {
				v.Data["preview"] = template.HTML(b.RenderTrustedMarkdown(post.Body))
			} else {
				v.Data["preview"] = template.HTML(post.Body)
			}
		case "post":
			if err := post.Validate(); err != nil {
				v.Error = err
			} else {
				author, _ := b.CurrentUser(r)
				post.AuthorID = author.ID

				post.Updated = time.Now().UTC()
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
