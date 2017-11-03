package models

import "github.com/kirsle/blog/core/jsondb"

// Model is a generic interface for models.
type Model interface {
	UseDB(*jsondb.DB)
}

// Base is an implementation of the Model interface suitable for including in
// your actual models.
type Base struct {
	DB *jsondb.DB
}

// UseDB stores a reference to your JSON DB for the model to use.
func (b *Base) UseDB(db *jsondb.DB) {
	b.DB = db
}
