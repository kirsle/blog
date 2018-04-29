package events

import (
	"html/template"
	"net/http"
	"strconv"

	"github.com/kirsle/blog/internal/markdown"
	"github.com/kirsle/blog/internal/render"
	"github.com/kirsle/blog/internal/responses"
	"github.com/kirsle/blog/models/events"
)

func editHandler(w http.ResponseWriter, r *http.Request) {
	v := map[string]interface{}{
		"preview": "",
	}
	var ev *events.Event

	// Are we editing an existing event?
	if idStr := r.FormValue("id"); idStr != "" {
		id, err := strconv.Atoi(idStr)
		if err == nil {
			ev, err = events.Load(id)
			if err != nil {
				responses.Flash(w, r, "That event ID was not found")
				ev = events.New()
			}
		}
	} else {
		ev = events.New()
	}

	if r.Method == http.MethodPost {
		// Parse from form values.
		ev.ParseForm(r)

		// Previewing, or submitting?
		switch r.FormValue("submit") {
		case "preview":
			v["preview"] = template.HTML(markdown.RenderTrustedMarkdown(ev.Description))
		case "save":
			if err := ev.Validate(); err != nil {
				responses.Flash(w, r, "Error: %s", err.Error())
			} else {
				err = ev.Save()

				if err != nil {
					responses.Flash(w, r, "Error: %s", err.Error())
				} else {
					responses.Flash(w, r, "Event created!")
					responses.Redirect(w, "/e/"+ev.Fragment)
				}
			}
		}
	}

	v["event"] = ev
	render.Template(w, r, "events/edit", v)
}
