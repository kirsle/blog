package authctl

import (
	"net/http"

	"github.com/kirsle/blog/internal/log"
	"github.com/kirsle/blog/internal/render"
	"github.com/kirsle/blog/internal/responses"
	"github.com/kirsle/blog/internal/sessions"
)

// AgeGate handles age verification for NSFW blogs.
func AgeGate(w http.ResponseWriter, r *http.Request) {
	next := r.FormValue("next")
	if next == "" {
		next = "/"
	}
	v := map[string]interface{}{
		"Next": next,
	}

	if r.Method == http.MethodPost {
		confirm := r.FormValue("confirm")
		log.Info("confirm: %s", confirm)
		if r.FormValue("confirm") == "true" {
			session := sessions.Get(r)
			session.Values["age-ok"] = true
			session.Save(r, w)
			responses.Redirect(w, next)
			return
		}
	}

	render.Template(w, r, ".age-gate.gohtml", v)
}
