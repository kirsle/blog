package core

import (
	"net/http"
	"path/filepath"

	"github.com/gorilla/mux"
	"github.com/kirsle/blog/core/jsondb"
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
	n *negroni.Negroni // Negroni middleware manager
	r *mux.Router      // Router
}

// New initializes the Blog application.
func New(documentRoot, userRoot string) *Blog {
	blog := &Blog{
		DocumentRoot: documentRoot,
		UserRoot:     userRoot,
		DB:           jsondb.New(filepath.Join(userRoot, ".private")),
	}

	r := mux.NewRouter()
	blog.r = r
	r.HandleFunc("/admin/setup", blog.SetupHandler)
	r.HandleFunc("/", blog.PageHandler)
	r.NotFoundHandler = http.HandlerFunc(blog.PageHandler)

	n := negroni.New(
		negroni.NewRecovery(),
		negroni.NewLogger(),
	)
	blog.n = n
	n.UseHandler(r)

	return blog
}

// ListenAndServe begins listening on the given bind address.
func (b *Blog) ListenAndServe(address string) {
	log.Info("Listening on %s", address)
	http.ListenAndServe(address, b.n)
}
