package admin

import (
	"net/http"
	"strconv"

	"github.com/kirsle/blog/core/internal/forms"
	"github.com/kirsle/blog/core/internal/models/settings"
	"github.com/kirsle/blog/core/internal/render"
	"github.com/kirsle/blog/core/internal/responses"
)

func settingsHandler(w http.ResponseWriter, r *http.Request) {
	// Get the current settings.
	settings, _ := settings.Load()
	v := map[string]interface{}{
		"s": settings,
	}

	if r.Method == http.MethodPost {
		redisPort, _ := strconv.Atoi(r.FormValue("redis-port"))
		redisDB, _ := strconv.Atoi(r.FormValue("redis-db"))
		mailPort, _ := strconv.Atoi(r.FormValue("mail-port"))
		form := &forms.Settings{
			Title:        r.FormValue("title"),
			Description:  r.FormValue("description"),
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
		settings.Site.Description = form.Description
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
			v["Error"] = err
		} else {
			// Save the settings.
			settings.Save()
			// b.Configure()

			responses.FlashAndReload(w, r, "Settings have been saved!")
			return
		}
	}
	render.Template(w, r, "admin/settings", v)
}
