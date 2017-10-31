// Package blog is a web application which lets you host your own web blog,
// photo albums, wiki, etc.
//
// It is currently under early development and is not yet stable.
package main

import (
	"flag"

	"github.com/kirsle/blog/core"
)

// Build-time config constants.
var (
	Version      = "0.0.1"
	Build        = "live"
	DocumentRoot = "root"
)

// Command line args.
var (
	fDebug   bool
	fAddress string
)

func init() {
	flag.BoolVar(&fDebug, "debug", false, "Debug mode")
	flag.BoolVar(&fDebug, "d", false, "Debug mode (alias)")
	flag.StringVar(&fAddress, "address", ":8000", "Bind address")
	flag.StringVar(&fAddress, "a", ":8000", "Bind address (alias)")
}

func main() {
	flag.Parse()

	app := core.New(DocumentRoot, "")
	app.Debug = fDebug
	app.ListenAndServe(fAddress)
}
