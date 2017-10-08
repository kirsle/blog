package core

import "net/http"

// Redirect sends an HTTP redirect response.
func Redirect(w http.ResponseWriter, location string) {
	w.Header().Set("Location", location)
	w.WriteHeader(http.StatusFound)
}
