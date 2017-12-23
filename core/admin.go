package core

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/kirsle/blog/core/caches/null"
	"github.com/kirsle/blog/core/caches/redis"
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
			RedisEnabled: len(r.FormValue("redis-enabled")) > 0,
			RedisHost:    r.FormValue("redis-host"),
			RedisPort:    redisPort,
			RedisDB:      redisDB,
			RedisPrefix:  r.FormValue("redis-prefix"),
			MailEnabled:  len(r.FormValue("mail-enabled")) > 0,
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

			// Reset Redis configuration.
			if settings.Redis.Enabled {
				cache, err := redis.New(
					fmt.Sprintf("%s:%d", settings.Redis.Host, settings.Redis.Port),
					settings.Redis.DB,
					settings.Redis.Prefix,
				)
				if err != nil {
					b.Flash(w, r, "Error connecting to Redis: %s", err)
					b.Cache = null.New()
				} else {
					b.Cache = cache
				}
			} else {
				b.Cache = null.New()
			}
			b.DB.Cache = b.Cache

			b.FlashAndReload(w, r, "Settings have been saved!")
			return
		}
	}
	b.RenderTemplate(w, r, "admin/settings", v)
}
