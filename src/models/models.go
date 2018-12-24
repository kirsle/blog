package models

import (
	"github.com/jinzhu/gorm"
	"github.com/kirsle/golog"
)

// DB is a reference to the parent app's gorm DB.
var DB *gorm.DB

// UseDB registers the DB from the root app.
func UseDB(db *gorm.DB) {
	DB = db
	DB.AutoMigrate(&Question{})
}

var log *golog.Logger

func init() {
	log = golog.GetLogger("blog")
}
