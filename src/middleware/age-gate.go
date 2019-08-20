package middleware

import (
	"net/http"
	"strings"

	"github.com/kirsle/blog/models/settings"
	"github.com/kirsle/blog/src/responses"
	"github.com/kirsle/blog/src/sessions"
	"github.com/urfave/negroni"
)

var ageGateSuffixes = []string{
	"/blog.rss", // Allow public access to RSS and Atom feeds.
	"/blog.atom",
	".js",
	".css",
	".txt",
	".ico",
	".png",
	".jpg",
	".jpeg",
	".gif",
	".mp4",
	".webm",
}

// AgeGate is a middleware generator that does age verification for NSFW sites.
func AgeGate(verifyHandler func(http.ResponseWriter, *http.Request)) negroni.HandlerFunc {
	middleware := func(w http.ResponseWriter, r *http.Request, next http.HandlerFunc) {
		s, _ := settings.Load()
		if !s.Site.NSFW {
			next(w, r)
			return
		}

		path := r.URL.Path
		if strings.HasPrefix(path, "/age-verify") {
			verifyHandler(w, r) // defer to the age gate handler itself.
			return
		}

		// Allow static files and things through.
		for _, suffix := range ageGateSuffixes {
			if strings.HasSuffix(path, suffix) {
				next(w, r)
				return
			}
		}

		// See if they've been cleared.
		session := sessions.Get(r)
		if val, _ := session.Values["age-ok"].(bool); !val {
			// They haven't been verified.
			// Allow single-page loads with ?over18=1 in query parameter.
			if r.FormValue("over18") == "" {
				responses.Redirect(w, "/age-verify?next="+r.URL.Path)
				return
			}
		}

		next(w, r)
	}

	return middleware
}
