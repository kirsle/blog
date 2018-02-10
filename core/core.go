// Package core implements the core source code of kirsle/blog.
package core

import (
	"fmt"
	"net/http"
	"path/filepath"

	"github.com/gorilla/mux"
	"github.com/gorilla/sessions"
	"github.com/kirsle/blog/core/internal/markdown"
	"github.com/kirsle/blog/core/internal/models/comments"
	"github.com/kirsle/blog/core/internal/models/posts"
	"github.com/kirsle/blog/core/internal/models/settings"
	"github.com/kirsle/blog/core/internal/models/users"
	"github.com/kirsle/blog/jsondb"
	"github.com/kirsle/blog/jsondb/caches"
	"github.com/kirsle/blog/jsondb/caches/null"
	"github.com/kirsle/blog/jsondb/caches/redis"
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

	DB    *jsondb.DB
	Cache caches.Cacher

	// Web app objects.
	n     *negroni.Negroni // Negroni middleware manager
	r     *mux.Router      // Router
	store sessions.Store
}

// New initializes the Blog application.
func New(documentRoot, userRoot string) *Blog {
	blog := &Blog{
		DocumentRoot: documentRoot,
		UserRoot:     userRoot,
		DB:           jsondb.New(filepath.Join(userRoot, ".private")),
		Cache:        null.New(),
	}

	// Load the site config, or start with defaults if not found.
	settings.DB = blog.DB
	config, err := settings.Load()
	if err != nil {
		config = settings.Defaults()
	}

	// Initialize the session cookie store.
	blog.store = sessions.NewCookieStore([]byte(config.Security.SecretKey))
	users.HashCost = config.Security.HashCost

	// Initialize the rest of the models.
	posts.DB = blog.DB
	users.DB = blog.DB
	comments.DB = blog.DB

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
			blog.Cache = cache
			blog.DB.Cache = cache
			markdown.Cache = cache
		}
	}

	// Initialize the router.
	r := mux.NewRouter()
	r.HandleFunc("/initial-setup", blog.SetupHandler)
	blog.AuthRoutes(r)
	blog.AdminRoutes(r)
	blog.ContactRoutes(r)
	blog.BlogRoutes(r)
	blog.CommentRoutes(r)

	// GitHub Flavored Markdown CSS.
	r.Handle("/css/gfm.css", http.StripPrefix("/css", http.FileServer(gfmstyle.Assets)))

	r.PathPrefix("/").HandlerFunc(blog.PageHandler)
	r.NotFoundHandler = http.HandlerFunc(blog.PageHandler)

	n := negroni.New(
		negroni.NewRecovery(),
		negroni.NewLogger(),
		negroni.HandlerFunc(blog.SessionLoader),
		negroni.HandlerFunc(blog.CSRFMiddleware),
		negroni.HandlerFunc(blog.AuthMiddleware),
	)
	n.UseHandler(r)

	// Keep references handy elsewhere in the app.
	blog.n = n
	blog.r = r

	return blog
}

// ListenAndServe begins listening on the given bind address.
func (b *Blog) ListenAndServe(address string) {
	log.Info("Listening on %s", address)
	http.ListenAndServe(address, b.n)
}
