package events

import (
	"net/http"
	"sort"

	"github.com/gorilla/mux"
	"github.com/kirsle/blog/internal/log"
	"github.com/kirsle/blog/internal/middleware/auth"
	"github.com/kirsle/blog/internal/render"
	"github.com/kirsle/blog/internal/responses"
	"github.com/kirsle/blog/models/events"
	"github.com/urfave/negroni"
)

// Register the blog routes to the app.
func Register(r *mux.Router, loginError http.HandlerFunc) {
	// Login-required routers.
	loginRouter := mux.NewRouter()
	loginRouter.HandleFunc("/e/admin/edit", editHandler)
	loginRouter.HandleFunc("/e/admin/invite/{id}", inviteHandler)
	loginRouter.HandleFunc("/e/admin/", indexHandler)
	r.PathPrefix("/e/admin").Handler(
		negroni.New(
			negroni.HandlerFunc(auth.LoginRequired(loginError)),
			negroni.Wrap(loginRouter),
		),
	)

	// Public routes
	r.HandleFunc("/e/{fragment}", viewHandler)
}

// Admin index to view all events.
func indexHandler(w http.ResponseWriter, r *http.Request) {
	result := []*events.Event{}
	docs, _ := events.DB.List("events/by-id")
	for _, doc := range docs {
		ev := &events.Event{}
		err := events.DB.Get(doc, &ev)
		if err != nil {
			log.Error("error reading %s: %s", doc, err)
			continue
		}

		result = append(result, ev)
	}

	sort.Sort(sort.Reverse(events.ByDate(result)))

	render.Template(w, r, "events/index", map[string]interface{}{
		"events": result,
	})
}

// User handler to view a single event page.
func viewHandler(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	fragment, ok := params["fragment"]
	if !ok {
		responses.NotFound(w, r, "Not Found")
		return
	}

	event, err := events.LoadFragment(fragment)
	if err != nil {
		responses.FlashAndRedirect(w, r, "/", "Event Not Found")
		return
	}

	v := map[string]interface{}{
		"event": event,
	}
	render.Template(w, r, "events/view", v)
}
