package postctl

import (
	"net/http"
	"sort"
	"time"

	"github.com/kirsle/blog/internal/middleware/auth"
	"github.com/kirsle/blog/models/posts"
	"github.com/kirsle/blog/internal/render"
	"github.com/kirsle/blog/internal/responses"
	"github.com/kirsle/blog/internal/types"
)

// archiveHandler summarizes all blog entries in an archive view.
func archiveHandler(w http.ResponseWriter, r *http.Request) {
	idx, err := posts.GetIndex()
	if err != nil {
		responses.BadRequest(w, r, "Error getting blog index")
		return
	}

	// Group posts by calendar month.
	var months []string
	byMonth := map[string]*Archive{}
	for _, post := range idx.Posts {
		// Exclude certain posts
		if (post.Privacy == types.PRIVATE || post.Privacy == types.UNLISTED) && !auth.LoggedIn(r) {
			continue
		} else if post.Privacy == types.DRAFT {
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

	v := map[string]interface{}{
		"Archive": result,
	}
	render.Template(w, r, "blog/archive", v)
}
