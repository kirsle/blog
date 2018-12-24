package postctl

import (
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/feeds"
	"github.com/kirsle/blog/src/responses"
	"github.com/kirsle/blog/models/posts"
	"github.com/kirsle/blog/models/settings"
	"github.com/kirsle/blog/models/users"
)

func feedHandler(w http.ResponseWriter, r *http.Request) {
	config, _ := settings.Load()
	admin, err := users.Load(1)
	if err != nil {
		responses.Error(w, r, "Blog isn't ready yet.")
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
	for i, p := range RecentPosts(r, "", "") {
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
		if i == 9 { // 10 -1
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
