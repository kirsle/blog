package core

import (
	"net/http"
	"path/filepath"

	"github.com/gorilla/mux"
	"github.com/gorilla/sessions"
	"github.com/kirsle/blog/core/jsondb"
	"github.com/kirsle/blog/core/models/settings"
	"github.com/kirsle/blog/core/models/users"
	"github.com/urfave/negroni"
)

// Blog is the root application object that maintains the app configuration
// and helper objects.
type Blog struct {
	Debug bool

	// DocumentRoot is the core static files root; UserRoot masks over it.
	DocumentRoot string
	UserRoot     string

	DB *jsondb.DB

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
	users.DB = blog.DB

	// Initialize the router.
	r := mux.NewRouter()
	r.HandleFunc("/initial-setup", blog.SetupHandler)
	r.HandleFunc("/login", blog.LoginHandler)
	r.HandleFunc("/logout", blog.LogoutHandler)
	blog.AdminRoutes(r)

	r.PathPrefix("/").HandlerFunc(blog.PageHandler)
	r.NotFoundHandler = http.HandlerFunc(blog.PageHandler)

	n := negroni.New(
		negroni.NewRecovery(),
		negroni.NewLogger(),
		negroni.HandlerFunc(blog.SessionLoader),
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
