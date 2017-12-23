// Package blog is a web application which lets you host your own web blog,
// photo albums, wiki, etc.
//
// It is currently under early development and is not yet stable.
package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/kirsle/blog/core"
	"github.com/kirsle/blog/core/jsondb"
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
	userRoot := flag.Arg(0)
	if userRoot == "" {
		fmt.Printf("Need user root\n")
		os.Exit(1)
	}

	app := core.New(DocumentRoot, userRoot)
	if fDebug {
		app.Debug = true
	}

	// Set $JSONDB_DEBUG=1 to debug JsonDB; it's very noisy!
	if os.Getenv("JSONDB_DEBUG") != "" {
		jsondb.SetDebug(true)
	}

	app.ListenAndServe(fAddress)
}
