package admin

import (
	"net/http"

	"github.com/gorilla/mux"
	"github.com/kirsle/blog/src/middleware/auth"
	"github.com/kirsle/blog/src/render"
	"github.com/urfave/negroni"
)

// Register the initial setup routes.
func Register(r *mux.Router, authErrorFunc http.HandlerFunc) {
	adminRouter := mux.NewRouter().PathPrefix("/admin").Subrouter().StrictSlash(true)
	adminRouter.HandleFunc("/", indexHandler)
	adminRouter.HandleFunc("/settings", settingsHandler)
	adminRouter.HandleFunc("/editor", editorHandler)
	adminRouter.HandleFunc("/upload", uploadHandler)

	r.PathPrefix("/admin").Handler(negroni.New(
		negroni.HandlerFunc(auth.LoginRequired(authErrorFunc)),
		negroni.Wrap(adminRouter),
	))
}

func indexHandler(w http.ResponseWriter, r *http.Request) {
	render.Template(w, r, "admin/index", nil)
}
