package questions

import (
	"fmt"
	"html/template"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"github.com/kirsle/blog/models/posts"
	"github.com/kirsle/blog/models/settings"
	"github.com/kirsle/blog/src/log"
	"github.com/kirsle/blog/src/mail"
	"github.com/kirsle/blog/src/markdown"
	"github.com/kirsle/blog/src/middleware/auth"
	"github.com/kirsle/blog/src/models"
	"github.com/kirsle/blog/src/render"
	"github.com/kirsle/blog/src/responses"
	"github.com/kirsle/blog/src/sessions"
	"github.com/urfave/negroni"
)

var badRequest func(http.ResponseWriter, *http.Request, string)

// Register the comment routes to the app.
func Register(r *mux.Router, loginError http.HandlerFunc) {
	badRequest = responses.BadRequest

	r.HandleFunc("/ask", questionsHandler)
	r.Handle("/ask/answer",
		negroni.New(
			negroni.HandlerFunc(auth.LoginRequired(loginError)),
			negroni.WrapFunc(answerHandler),
		),
	).Methods(http.MethodPost)
}

func questionsHandler(w http.ResponseWriter, r *http.Request) {
	// Share their name and email with the commenting system.
	session := sessions.Get(r)
	name, _ := session.Values["c.name"].(string)
	email, _ := session.Values["c.email"].(string)

	Q := models.NewQuestion()
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
						markdown.RenderMarkdown(
							Q.Question +
								"\n\nAnswer this at " + strings.Trim(cfg.Site.URL, "/") + "/ask",
						),
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
	}

	v["Q"] = Q

	// Load the pending questions.
	pending, err := models.PendingQuestions(0, 20)
	if err != nil {
		log.Error(err.Error())
	}
	v["Pending"] = pending

	render.Template(w, r, "questions.gohtml", v)
}

func answerHandler(w http.ResponseWriter, r *http.Request) {
	submit := r.FormValue("submit")

	cfg, err := settings.Load()
	if err != nil {
		responses.Error(w, r, "Error loading site configuration!")
		return
	}

	type answerForm struct {
		ID     int
		Answer string
		Submit string
	}
	id, _ := strconv.Atoi(r.FormValue("id"))
	form := answerForm{
		ID:     id,
		Answer: r.FormValue("answer"),
		Submit: r.FormValue("submit"),
	}

	// Look up the question.
	Q, err := models.GetQuestion(form.ID)
	if err != nil {
		responses.FlashAndRedirect(w, r, "/ask",
			fmt.Sprintf("Did not find question ID %d", form.ID),
		)
		return
	}

	switch submit {
	case "answer":
		// Prepare a Markdown-themed blog post and go to the Preview page for it.
		blog := posts.New()
		blog.Title = "Ask"
		blog.Tags = []string{"ask"}
		blog.Fragment = fmt.Sprintf("ask-%s",
			time.Now().Format("20060102150405"),
		)
		blog.Body = fmt.Sprintf(
			"> **%s** asks:\n>\n> %s\n\n"+
				"%s\n",
			Q.Name,
			strings.Replace(Q.Question, "\n", "> \n", 0),
			form.Answer,
		)

		Q.Status = models.Answered
		Q.Save()

		// TODO: email the person who asked about the new URL.
		if Q.Email != "" {
			log.Info("Notifying user %s by email that the question is answered", Q.Email)
			go mail.SendEmail(mail.Email{
				To:       Q.Email,
				Subject:  "Your question has been answered",
				Template: ".email/generic.gohtml",
				Data: map[string]interface{}{
					"Subject": "Your question has been answered",
					"Message": template.HTML(
						markdown.RenderMarkdown(
							fmt.Sprintf(
								"Hello, %s\n\n"+
									"Your recent question on %s has been answered. To "+
									"view the answer, please visit the following link:\n\n"+
									"%s/%s",
								Q.Name,
								cfg.Site.Title,
								cfg.Site.URL,
								blog.Fragment,
							),
						),
					),
				},
			})
		}

		render.Template(w, r, "blog/edit", map[string]interface{}{
			"preview": template.HTML(markdown.RenderTrustedMarkdown(blog.Body)),
			"post":    blog,
		})
		return
	case "delete":
		Q.Status = models.Deleted
		Q.Save()
		responses.FlashAndRedirect(w, r, "/ask", "Question deleted.")
		return
	default:
		responses.FlashAndRedirect(w, r, "/ask", "Unknown submit action.")
		return
	}
}
