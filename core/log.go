package core

import "github.com/kirsle/golog"

var log *golog.Logger

func init() {
	log = golog.GetLogger("blog")
	log.Configure(&golog.Config{
		Colors: golog.ExtendedColor,
		Theme:  golog.DarkTheme,
	})
}
