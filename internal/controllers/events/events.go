package events

import (
	"fmt"
	"net/http"
	"sort"

	"github.com/gorilla/mux"
	"github.com/kirsle/blog/internal/log"
	"github.com/kirsle/blog/internal/middleware/auth"
	"github.com/kirsle/blog/internal/render"
	"github.com/kirsle/blog/internal/responses"
	"github.com/kirsle/blog/models/comments"
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
	r.HandleFunc("/c/logout", contactLogoutHandler)
	r.HandleFunc("/c/{secret}", contactAuthHandler)
}

// Admin index to view all events.
func indexHandler(w http.ResponseWriter, r *http.Request) {
	result, err := events.All()
	if err != nil {
		log.Error("error listing all events: %s", err)
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

	// Template variables.
	v := map[string]interface{}{
		"event":      event,
		"authedRSVP": events.RSVP{},
	}

	// Sort the guest list.
	sort.Sort(events.ByName(event.RSVP))

	// Is the browser session authenticated as a contact?
	authedContact, err := AuthedContact(r)
	if err == nil {
		v["authedContact"] = authedContact

		// Do they have an RSVP?
		for _, rsvp := range event.RSVP {
			if rsvp.ContactID == authedContact.ID {
				v["authedRSVP"] = rsvp
				break
			}
		}
	}

	// Count up the RSVP statuses, and also look for the authed contact's RSVP.
	var (
		countGoing    int
		countMaybe    int
		countNotGoing int
		countInvited  int
	)
	for _, rsvp := range event.RSVP {
		if authedContact.ID != 0 && rsvp.ContactID == authedContact.ID {
			v["authedRVSP"] = rsvp
		}

		switch rsvp.Status {
		case events.StatusGoing:
			countGoing++
		case events.StatusMaybe:
			countMaybe++
		case events.StatusNotGoing:
			countNotGoing++
		default:
			countInvited++
		}
	}
	v["countGoing"] = countGoing
	v["countMaybe"] = countMaybe
	v["countNotGoing"] = countNotGoing
	v["countInvited"] = countInvited

	// If we're posting, are we RSVPing?
	if r.Method == http.MethodPost {
		action := r.PostFormValue("action")
		switch action {
		case "answer-rsvp":
			// Subscribe them to the comment thread on this page if we have an email.
			if authedContact.Email != "" {
				thread := fmt.Sprintf("event-%d", event.ID)
				log.Info("events.viewHandler: subscribe email %s to thread %s", authedContact.Email, thread)
				ml := comments.LoadMailingList()
				ml.Subscribe(thread, authedContact.Email)
			}

			answer := r.PostFormValue("submit")
			for _, rsvp := range event.RSVP {
				if rsvp.ContactID == authedContact.ID {
					log.Info("Mark RSVP status %s for contact %s", answer, authedContact.Name())
					rsvp.Status = answer
					rsvp.Save()
					responses.FlashAndReload(w, r, "You have confirmed '%s' for your RSVP.", answer)
					return
				}
			}
		default:
			responses.FlashAndReload(w, r, "Invalid form action.")
		}
		responses.FlashAndReload(w, r, "Unknown error.")
		return
	}

	render.Template(w, r, "events/view", v)
}
