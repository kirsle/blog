package core

import (
	"html/template"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/kirsle/blog/core/models/posts"
	"github.com/urfave/negroni"
)

// BlogRoutes attaches the blog routes to the app.
func (b *Blog) BlogRoutes(r *mux.Router) {
	// Login-required routers.
	loginRouter := mux.NewRouter()
	loginRouter.HandleFunc("/blog/edit", b.EditBlog)
	r.PathPrefix("/blog").Handler(
		negroni.New(
			negroni.HandlerFunc(b.LoginRequired),
			negroni.Wrap(loginRouter),
		),
	)

	adminRouter := mux.NewRouter().PathPrefix("/admin").Subrouter().StrictSlash(false)
	r.HandleFunc("/admin", b.AdminHandler) // so as to not be "/admin/"
	adminRouter.HandleFunc("/settings", b.SettingsHandler)
	adminRouter.PathPrefix("/").HandlerFunc(b.PageHandler)
	r.PathPrefix("/admin").Handler(negroni.New(
		negroni.HandlerFunc(b.LoginRequired),
		negroni.Wrap(adminRouter),
	))
}

// EditBlog is the blog writing and editing page.
func (b *Blog) EditBlog(w http.ResponseWriter, r *http.Request) {
	v := NewVars(map[interface{}]interface{}{
		"preview": "",
	})
	post := posts.New()

	if r.Method == http.MethodPost {
		// Parse from form values.
		post.LoadForm(r)

		// Previewing, or submitting?
		switch r.FormValue("submit") {
		case "preview":
			v.Data["preview"] = template.HTML(b.RenderMarkdown(post.Body))
		case "submit":
			if err := post.Validate(); err != nil {
				v.Error = err
			}
		}
	}

	v.Data["post"] = post
	b.RenderTemplate(w, r, "blog/edit", v)
}
