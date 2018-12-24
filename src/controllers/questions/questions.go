package questions

import (
	"fmt"
	"html/template"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"github.com/kirsle/blog/src/log"
	"github.com/kirsle/blog/src/mail"
	"github.com/kirsle/blog/src/markdown"
	"github.com/kirsle/blog/src/render"
	"github.com/kirsle/blog/src/responses"
	"github.com/kirsle/blog/src/sessions"
	"github.com/kirsle/blog/models/comments"
	"github.com/kirsle/blog/models/questions"
	"github.com/kirsle/blog/models/settings"
	"github.com/kirsle/blog/models/users"
)

var badRequest func(http.ResponseWriter, *http.Request, string)

// Register the comment routes to the app.
func Register(r *mux.Router) {
	badRequest = responses.BadRequest

	r.HandleFunc("/ask", questionsHandler)
}

// CommentMeta is the template variables for comment threads.
type CommentMeta struct {
	NewComment comments.Comment
	ID         string
	OriginURL  string // URL where original comment thread appeared
	Subject    string // email subject
	Thread     *comments.Thread
	Authors    map[int]*users.User
	CSRF       string
}

func questionsHandler(w http.ResponseWriter, r *http.Request) {
	submit := r.FormValue("submit")

	// Share their name and email with the commenting system.
	session := sessions.Get(r)
	name, _ := session.Values["c.name"].(string)
	email, _ := session.Values["c.email"].(string)

	Q := questions.New()
	Q.Name = name
	Q.Email = email

	cfg, err := settings.Load()
	if err != nil {
		responses.Error(w, r, "Error loading site configuration!")
		return
	}

	v := map[string]interface{}{}

	// Previewing, deleting, or posting?
	if r.Method == http.MethodPost {
		Q.ParseForm(r)
		log.Info("Q: %+v", Q)

		switch submit {
		case "ask":
			if err := Q.Validate(); err != nil {
				log.Debug("Validation error on question form: %s", err.Error())
				v["Error"] = err
			} else {
				// Cache their name and email in their session.
				session.Values["c.name"] = Q.Name
				session.Values["c.email"] = Q.Email
				session.Save(r, w)

				// Append their comment.
				err := Q.Save()
				if err != nil {
					log.Error("Error saving new question: %s", err.Error())
					responses.FlashAndRedirect(w, r, "/ask", "Error saving question: %s", err)
					return
				}

				// Email the site admin.
				subject := fmt.Sprintf("Ask Me Anything (%s) from %s", cfg.Site.Title, Q.Name)
				log.Info("Emailing site admin about this question")
				go mail.SendEmail(mail.Email{
					To:       cfg.Site.AdminEmail,
					Admin:    true,
					ReplyTo:  Q.Email,
					Subject:  subject,
					Template: ".email/generic.gohtml",
					Data: map[string]interface{}{
						"Subject": subject,
						"Message": template.HTML(
							markdown.RenderMarkdown(Q.Question) + "\n\n" +
								"Answer this at " + strings.Trim(cfg.Site.URL, "/") + "/ask",
						),
					},
				})

				// Log it to disk, too.
				fh, err := os.OpenFile(filepath.Join(*render.UserRoot, ".questions.log"), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
				if err != nil {
					responses.Flash(w, r, "Error logging the message to disk: %s", err)
				} else {
					fh.WriteString(fmt.Sprintf(
						"Date: %s\nName: %s\nEmail: %s\n\n%s\n\n--------------------\n\n",
						time.Now().Format(time.UnixDate),
						Q.Name,
						Q.Email,
						Q.Question,
					))
					fh.Close()
				}

				log.Info("Recorded question from %s: %s", Q.Name, Q.Question)
				responses.FlashAndRedirect(w, r, "/ask", "Your question has been recorded!")
				return
			}
		case "answer":
		case "delete":
		default:
			responses.FlashAndRedirect(w, r, "/ask", "Unknown submit action.")
			return
		}
	}

	v["Q"] = Q

	render.Template(w, r, "questions.gohtml", v)
}
