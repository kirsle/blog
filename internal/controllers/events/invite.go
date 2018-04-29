package events

import (
	"net/http"
	"strconv"

	"github.com/kirsle/blog/internal/render"
	"github.com/kirsle/blog/internal/responses"
	"github.com/kirsle/blog/models/events"
)

func inviteHandler(w http.ResponseWriter, r *http.Request) {
	v := map[string]interface{}{
		"preview": "",
	}

	id, err := strconv.Atoi(r.FormValue("id"))
	if err != nil {
		responses.FlashAndRedirect(w, r, "/e/admin/", "Invalid ID")
		return
	}
	event, err := events.Load(id)
	if err != nil {
		responses.FlashAndRedirect(w, r, "/e/admin/", "Can't load event: %s", err)
		return
	}

	v["event"] = event
	render.Template(w, r, "events/invite", v)
}
