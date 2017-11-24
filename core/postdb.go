package core

import (
	"github.com/kirsle/blog/core/jsondb"
)

// PostHelper is a singleton helper to manage the database controls for blog
// entries.
type PostHelper struct {
	master *Blog
	DB     *jsondb.DB
}

// InitPostHelper initializes the blog post controller helper.
func InitPostHelper(master *Blog) *PostHelper {
	return &PostHelper{
		master: master,
		DB:     master.DB,
	}
}

// GetIndex loads the blog index (cache).
func (p *PostHelper) GetIndex() {}
