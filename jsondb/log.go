// Package jsondb implements a flat file JSON database engine.
package jsondb

import (
	"github.com/kirsle/golog"
)

var log *golog.Logger

func init() {
	log = golog.GetLogger("jsondb")
	log.Configure(&golog.Config{
		Level:  golog.InfoLevel,
		Colors: golog.ExtendedColor,
		Theme:  golog.DarkTheme,
	})
}

// SetDebug turns on debug logging.
func SetDebug(debug bool) {
	if debug {
		log.Config.Level = golog.DebugLevel
		log.Debug("JsonDB Debug log enabled.")
	} else {
		log.Config.Level = golog.InfoLevel
		log.Info("JsonDB Debug log disabled.")
	}
}
