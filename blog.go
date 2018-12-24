// Package blog is a personal website and blogging app.
package blog

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	"github.com/gorilla/mux"
	"github.com/jinzhu/gorm"
	"github.com/kirsle/blog/jsondb"
	"github.com/kirsle/blog/jsondb/caches"
	"github.com/kirsle/blog/jsondb/caches/null"
	"github.com/kirsle/blog/jsondb/caches/redis"
	"github.com/kirsle/blog/models/comments"
	"github.com/kirsle/blog/models/posts"
	"github.com/kirsle/blog/models/settings"
	"github.com/kirsle/blog/models/users"
	"github.com/kirsle/blog/src/controllers/admin"
	"github.com/kirsle/blog/src/controllers/authctl"
	commentctl "github.com/kirsle/blog/src/controllers/comments"
	"github.com/kirsle/blog/src/controllers/contact"
	postctl "github.com/kirsle/blog/src/controllers/posts"
	questionsctl "github.com/kirsle/blog/src/controllers/questions"
	"github.com/kirsle/blog/src/controllers/setup"
	"github.com/kirsle/blog/src/log"
	"github.com/kirsle/blog/src/markdown"
	"github.com/kirsle/blog/src/middleware"
	"github.com/kirsle/blog/src/middleware/auth"
	"github.com/kirsle/blog/src/models"
	"github.com/kirsle/blog/src/render"
	"github.com/kirsle/blog/src/responses"
	"github.com/kirsle/blog/src/sessions"
	"github.com/shurcooL/github_flavored_markdown/gfmstyle"
	"github.com/urfave/negroni"
)

// Blog is the root application object that maintains the app configuration
// and helper objects.
type Blog struct {
	Debug bool

	// DocumentRoot is the core static files root; UserRoot masks over it.
	DocumentRoot string
	UserRoot     string

	db     *gorm.DB
	jsonDB *jsondb.DB
	Cache  caches.Cacher

	// Web app objects.
	n *negroni.Negroni // Negroni middleware manager
	r *mux.Router      // Router
}

// New initializes the Blog application.
func New(documentRoot, userRoot string) *Blog {
	// Initialize the SQLite database.
	if _, err := os.Stat(filepath.Join(userRoot, ".private")); os.IsNotExist(err) {
		os.MkdirAll(filepath.Join(userRoot, ".private"), 0755)
	}
	db, err := gorm.Open("sqlite3", filepath.Join(userRoot, ".private", "database.sqlite"))
	if err != nil {
		panic(err)
	}

	return &Blog{
		DocumentRoot: documentRoot,
		UserRoot:     userRoot,
		db:           db,
		jsonDB:       jsondb.New(filepath.Join(userRoot, ".private")),
		Cache:        null.New(),
	}
}

// Run quickly configures and starts the HTTP server.
func (b *Blog) Run(address string) {
	b.Configure()
	b.SetupHTTP()
	b.ListenAndServe(address)
}

// Configure initializes (or reloads) the blog's configuration, and binds the
// settings in sub-packages.
func (b *Blog) Configure() {
	if b.Debug {
		b.db.LogMode(true)
	}
	// Load the site config, or start with defaults if not found.
	settings.DB = b.jsonDB
	config, err := settings.Load()
	if err != nil {
		config = settings.Defaults()
	}

	// Bind configs in sub-packages.
	render.UserRoot = &b.UserRoot
	render.DocumentRoot = &b.DocumentRoot

	// Initialize the session cookie store.
	sessions.SetSecretKey([]byte(config.Security.SecretKey))
	users.HashCost = config.Security.HashCost

	// Initialize the rest of the models.
	posts.DB = b.jsonDB
	users.DB = b.jsonDB
	comments.DB = b.jsonDB
	models.UseDB(b.db)

	// Redis cache?
	if config.Redis.Enabled {
		addr := fmt.Sprintf("%s:%d", config.Redis.Host, config.Redis.Port)
		log.Info("Connecting to Redis at %s/%d", addr, config.Redis.DB)
		cache, err := redis.New(
			addr,
			config.Redis.DB,
			config.Redis.Prefix,
		)
		if err != nil {
			log.Error("Redis init error: %s", err.Error())
		} else {
			b.Cache = cache
			b.jsonDB.Cache = cache
			markdown.Cache = cache
		}
	}

	b.registerErrors()
}

// SetupHTTP initializes the Negroni middleware engine and registers routes.
func (b *Blog) SetupHTTP() {
	// Initialize the router.
	r := mux.NewRouter()
	setup.Register(r)
	authctl.Register(r)
	admin.Register(r, b.MustLogin)
	contact.Register(r)
	postctl.Register(r, b.MustLogin)
	commentctl.Register(r)
	questionsctl.Register(r, b.MustLogin)

	// GitHub Flavored Markdown CSS.
	r.Handle("/css/gfm.css", http.StripPrefix("/css", http.FileServer(gfmstyle.Assets)))

	r.PathPrefix("/").HandlerFunc(b.PageHandler)
	r.NotFoundHandler = http.HandlerFunc(b.PageHandler)

	n := negroni.New(
		negroni.NewRecovery(),
		negroni.NewLogger(),
		negroni.HandlerFunc(sessions.Middleware),
		negroni.HandlerFunc(middleware.CSRF(responses.Forbidden)),
		negroni.HandlerFunc(auth.Middleware),
		negroni.HandlerFunc(middleware.AgeGate(authctl.AgeGate)),
	)
	n.UseHandler(r)

	// Keep references handy elsewhere in the app.
	b.n = n
	b.r = r
}

// ListenAndServe begins listening on the given bind address.
func (b *Blog) ListenAndServe(address string) {
	log.Info("Listening on %s", address)
	http.ListenAndServe(address, b.n)
}

// MustLogin handles errors from the LoginRequired middleware by redirecting
// the user to the login page.
func (b *Blog) MustLogin(w http.ResponseWriter, r *http.Request) {
	responses.Redirect(w, "/login?next="+r.URL.Path)
}
