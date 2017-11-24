package core

import "regexp"

var (
	// CSS classes for Markdown fenced code blocks
	reFencedCodeClass = regexp.MustCompile("^highlight highlight-[a-zA-Z0-9]+$")
)
