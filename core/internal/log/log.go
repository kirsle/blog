// Package log implements the common logging engine for the blog.
package log

import "github.com/kirsle/golog"

var log *golog.Logger

func init() {
	log = golog.GetLogger("blog")
	log.Configure(&golog.Config{
		Colors: golog.ExtendedColor,
		Theme:  golog.DarkTheme,
	})
}

func Debug(m string, v ...interface{}) {
	log.Debug(m, v...)
}

func Info(m string, v ...interface{}) {
	log.Info(m, v...)
}

func Warn(m string, v ...interface{}) {
	log.Warn(m, v...)
}

func Error(m string, v ...interface{}) {
	log.Error(m, v...)
}
