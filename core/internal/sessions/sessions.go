package sessions

import (
	"context"
	"net/http"
	"time"

	"github.com/gorilla/sessions"
	"github.com/kirsle/blog/core/internal/log"
	"github.com/kirsle/blog/core/internal/types"
)

// Store holds your cookie store information.
var Store sessions.Store

// SetSecretKey initializes a session cookie store with the secret key.
func SetSecretKey(keyPairs ...[]byte) {
	Store = sessions.NewCookieStore(keyPairs...)
}

// Middleware gets the Gorilla session store and makes it available on the
// Request context.
//
// Middleware is the first custom middleware applied, so it takes the current
// datetime to make available later in the request and stores it on the request
// context.
func Middleware(w http.ResponseWriter, r *http.Request, next http.HandlerFunc) {
	// Store the current datetime on the request context.
	ctx := context.WithValue(r.Context(), types.StartTimeKey, time.Now())

	// Get the Gorilla session and make it available in the request context.
	session, _ := Store.Get(r, "session")
	ctx = context.WithValue(ctx, types.SessionKey, session)

	next(w, r.WithContext(ctx))
}

// Get returns the current request's session.
func Get(r *http.Request) *sessions.Session {
	if r == nil {
		panic("Session(*http.Request) with a nil argument!?")
	}

	ctx := r.Context()
	if session, ok := ctx.Value(types.SessionKey).(*sessions.Session); ok {
		return session
	}

	// If the session wasn't on the request, it means I broke something.
	log.Error(
		"Session(): didn't find session in request context! Getting it " +
			"from the session store instead.",
	)
	session, _ := Store.Get(r, "session")
	return session
}
