package core

import (
	"net/http"

	"github.com/gorilla/mux"
	"github.com/kirsle/blog/core/models/settings"
	"github.com/urfave/negroni"
)

// AdminRoutes attaches the admin routes to the app.
func (b *Blog) AdminRoutes(r *mux.Router) {
	adminRouter := mux.NewRouter().PathPrefix("/admin").Subrouter().StrictSlash(false)
	r.HandleFunc("/admin", b.AdminHandler) // so as to not be "/admin/"
	adminRouter.HandleFunc("/settings", b.SettingsHandler)
	adminRouter.PathPrefix("/").HandlerFunc(b.PageHandler)
	r.PathPrefix("/admin").Handler(negroni.New(
		negroni.HandlerFunc(b.LoginRequired),
		negroni.Wrap(adminRouter),
	))
}

// AdminHandler is the admin landing page.
func (b *Blog) AdminHandler(w http.ResponseWriter, r *http.Request) {
	b.RenderTemplate(w, r, "admin/index", nil)
}

// SettingsHandler lets you configure the app from the frontend.
func (b *Blog) SettingsHandler(w http.ResponseWriter, r *http.Request) {
	v := NewVars()

	// Get the current settings.
	settings, _ := settings.Load()
	v.Data["s"] = settings
	b.RenderTemplate(w, r, "admin/settings", v)
}
