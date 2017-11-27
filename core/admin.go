package core

import (
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/kirsle/blog/core/forms"
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

	if r.Method == http.MethodPost {
		redisPort, _ := strconv.Atoi(r.FormValue("redis-port"))
		redisDB, _ := strconv.Atoi(r.FormValue("redis-db"))
		mailPort, _ := strconv.Atoi(r.FormValue("mail-port"))
		form := &forms.Settings{
			Title:        r.FormValue("title"),
			AdminEmail:   r.FormValue("admin-email"),
			URL:          r.FormValue("url"),
			RedisEnabled: r.FormValue("redis-enabled") == "true",
			RedisHost:    r.FormValue("redis-host"),
			RedisPort:    redisPort,
			RedisDB:      redisDB,
			RedisPrefix:  r.FormValue("redis-prefix"),
			MailEnabled:  r.FormValue("mail-enabled") == "true",
			MailSender:   r.FormValue("mail-sender"),
			MailHost:     r.FormValue("mail-host"),
			MailPort:     mailPort,
			MailUsername: r.FormValue("mail-username"),
			MailPassword: r.FormValue("mail-password"),
		}

		// Copy form values into the settings struct for display, in case of
		// any validation errors.
		settings.Site.Title = form.Title
		settings.Site.AdminEmail = form.AdminEmail
		settings.Site.URL = form.URL
		settings.Redis.Enabled = form.RedisEnabled
		settings.Redis.Host = form.RedisHost
		settings.Redis.Port = form.RedisPort
		settings.Redis.DB = form.RedisDB
		settings.Redis.Prefix = form.RedisPrefix
		settings.Mail.Enabled = form.MailEnabled
		settings.Mail.Sender = form.MailSender
		settings.Mail.Host = form.MailHost
		settings.Mail.Port = form.MailPort
		settings.Mail.Username = form.MailUsername
		settings.Mail.Password = form.MailPassword
		err := form.Validate()
		if err != nil {
			v.Error = err
		} else {
			// Save the settings.
			settings.Save()
			b.FlashAndReload(w, r, "Settings have been saved!")
			return
		}
	}
	b.RenderTemplate(w, r, "admin/settings", v)
}
