package postctl

import (
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/gorilla/feeds"
	"github.com/kirsle/blog/models/posts"
	"github.com/kirsle/blog/models/settings"
	"github.com/kirsle/blog/models/users"
	"github.com/kirsle/blog/src/markdown"
	"github.com/kirsle/blog/src/responses"
	"github.com/kirsle/blog/src/types"
)

// Feed configuration. TODO make configurable.
var (
	FeedPostsPerPage = 20

	reRelativeLink = regexp.MustCompile(` (src|href|poster)=(['"])/([^'"]+)['"]`)
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

		// Render the post to HTML.
		var rendered string
		if post.ContentType == string(types.MARKDOWN) {
			rendered = markdown.RenderTrustedMarkdown(post.Body)
		} else {
			rendered = post.Body
		}

		// Make relative links absolute.
		matches := reRelativeLink.FindAllStringSubmatch(rendered, -1)
		for _, match := range matches {
			var (
				attr   = match[1]
				quote  = match[2]
				uri    = match[3]
				absURI = config.Site.URL + "/" + uri
				new    = fmt.Sprintf(" %s%s%s%s",
					attr, quote, absURI, quote,
				)
			)
			rendered = strings.Replace(rendered, match[0], new, 1)
		}

		feed.Items = append(feed.Items, &feeds.Item{
			Title:       p.Title,
			Link:        &feeds.Link{Href: config.Site.URL + p.Fragment},
			Description: rendered,
			Created:     p.Created,
		})
		if i == FeedPostsPerPage-1 {
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
		w.Header().Set("Content-Type", "application/rss+xml; encoding=utf-8")
		w.Write([]byte(rss))
	}
}
