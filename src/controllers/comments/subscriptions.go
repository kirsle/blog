package comments

import (
	"net/http"
	"net/mail"

	"github.com/kirsle/blog/models/comments"
	"github.com/kirsle/blog/src/render"
	"github.com/kirsle/blog/src/responses"
)

func subscriptionHandler(w http.ResponseWriter, r *http.Request) {
	// POST to unsubscribe from all threads.
	if r.Method == http.MethodPost {
		email := r.FormValue("email")
		if email == "" {
			badRequest(w, r, "email address is required to unsubscribe from comment threads")
		} else if _, err := mail.ParseAddress(email); err != nil {
			badRequest(w, r, "invalid email address")
		}

		m := comments.LoadMailingList()
		m.UnsubscribeAll(email)
		responses.FlashAndRedirect(w, r, "/comments/subscription",
			"You have been unsubscribed from all mailing lists.",
		)
		return
	}

	// GET to unsubscribe from a single thread.
	thread := r.URL.Query().Get("t")
	email := r.URL.Query().Get("e")
	if thread != "" && email != "" {
		m := comments.LoadMailingList()
		m.Unsubscribe(thread, email)
		responses.FlashAndRedirect(w, r, "/comments/subscription", "You have been unsubscribed successfully.")
		return
	}

	render.Template(w, r, "comments/subscription.gohtml", nil)
}
