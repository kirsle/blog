package core

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/gorilla/mux"
	"github.com/kirsle/blog/core/caches/null"
	"github.com/kirsle/blog/core/caches/redis"
	"github.com/kirsle/blog/core/forms"
	"github.com/kirsle/blog/core/models/settings"
	"github.com/urfave/negroni"
)

// AdminRoutes attaches the admin routes to the app.
func (b *Blog) AdminRoutes(r *mux.Router) {
	adminRouter := mux.NewRouter().PathPrefix("/admin").Subrouter().StrictSlash(true)
	adminRouter.HandleFunc("/", b.AdminHandler)
	adminRouter.HandleFunc("/settings", b.SettingsHandler)
	adminRouter.HandleFunc("/editor", b.EditorHandler)
	// r.HandleFunc("/admin", b.AdminHandler)
	r.PathPrefix("/admin").Handler(negroni.New(
		negroni.HandlerFunc(b.LoginRequired),
		negroni.Wrap(adminRouter),
	))
}

// AdminHandler is the admin landing page.
func (b *Blog) AdminHandler(w http.ResponseWriter, r *http.Request) {
	b.RenderTemplate(w, r, "admin/index", nil)
}

// FileTree holds information about files in the document roots.
type FileTree struct {
	UserRoot bool // false = CoreRoot
	Files    []Filepath
}

// EditorHandler lets you edit web pages from the frontend.
func (b *Blog) EditorHandler(w http.ResponseWriter, r *http.Request) {
	// Editing a page?
	file := strings.Trim(r.FormValue("file"), "/")
	if len(file) > 0 {
		var (
			fp       string
			fromCore = r.FormValue("from") == "core"
			saving   = r.FormValue("action") == "save"
			deleting = r.FormValue("action") == "delete"
			body     = []byte{}
		)

		// Are they saving?
		if saving {
			fp = filepath.Join(b.UserRoot, file)
			body = []byte(r.FormValue("body"))
			err := ioutil.WriteFile(fp, body, 0644)
			if err != nil {
				b.Flash(w, r, "Error saving: %s", err)
			} else {
				b.FlashAndRedirect(w, r, "/admin/editor?file="+url.QueryEscape(file), "Page saved successfully!")
				return
			}
		} else if deleting {
			fp = filepath.Join(b.UserRoot, file)
			err := os.Remove(fp)
			if err != nil {
				b.FlashAndRedirect(w, r, "/admin/editor", "Error deleting: %s", err)
			} else {
				b.FlashAndRedirect(w, r, "/admin/editor", "Page deleted!")
				return
			}
		} else {
			// Where is the file from?
			if fromCore {
				fp = filepath.Join(b.DocumentRoot, file)
			} else {
				fp = filepath.Join(b.UserRoot, file)
			}

			// Check the file. If not found, check from the core root.
			f, err := os.Stat(fp)
			if os.IsNotExist(err) {
				fp = filepath.Join(b.DocumentRoot, file)
				fromCore = true
				f, err = os.Stat(fp)
			}

			// If it exists, load it.
			if !os.IsNotExist(err) && !f.IsDir() {
				body, err = ioutil.ReadFile(fp)
				if err != nil {
					b.Flash(w, r, "Error reading %s: %s", fp, err)
				}
			}

			// Default HTML boilerplate for .gohtml templates.
			if len(body) == 0 && strings.HasSuffix(fp, ".gohtml") {
				body = []byte("{{ define \"title\" }}Untitled Page{{ end }}\n" +
					"{{ define \"content\" }}\n<h1>Untitled Page</h1>\n\n{{ end }}")
			}
		}

		v := NewVars(map[interface{}]interface{}{
			"File":     file,
			"Path":     fp,
			"Body":     string(body),
			"FromCore": fromCore,
		})
		b.RenderTemplate(w, r, "admin/editor", v)
		return
	}

	// Otherwise listing the index view.
	b.editorFileList(w, r)
}

// editorFileList handles the index view of /admin/editor.
func (b *Blog) editorFileList(w http.ResponseWriter, r *http.Request) {
	// Listing the file tree?
	trees := []FileTree{}
	for i, root := range []string{b.UserRoot, b.DocumentRoot} {
		tree := FileTree{
			UserRoot: i == 0,
			Files:    []Filepath{},
		}

		filepath.Walk(root, func(path string, f os.FileInfo, err error) error {
			abs, _ := filepath.Abs(path)
			rel, _ := filepath.Rel(root, path)

			// Skip hidden files and directories.
			if f.IsDir() || rel == "." || strings.HasPrefix(rel, ".private") || strings.HasPrefix(rel, "admin/") {
				return nil
			}

			// Only text files.
			ext := strings.ToLower(filepath.Ext(path))
			okTypes := []string{
				".html", ".gohtml", ".md", ".markdown", ".js", ".css", ".jsx",
			}
			ok := false
			for _, ft := range okTypes {
				if ext == ft {
					ok = true
					break
				}
			}
			if !ok {
				return nil
			}

			tree.Files = append(tree.Files, Filepath{
				Absolute: abs,
				Relative: rel,
				Basename: filepath.Base(path),
			})
			return nil
		})

		trees = append(trees, tree)
	}
	v := NewVars(map[interface{}]interface{}{
		"FileTrees": trees,
	})
	b.RenderTemplate(w, r, "admin/filelist", v)
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
