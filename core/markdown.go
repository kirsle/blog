package core

import (
	"github.com/microcosm-cc/bluemonday"
	"github.com/shurcooL/github_flavored_markdown"
)

// RenderMarkdown renders markdown to HTML, safely. It uses blackfriday to
// render Markdown to HTML and then Bluemonday to sanitize the resulting HTML.
func (b *Blog) RenderMarkdown(input string) string {
	unsafe := []byte(b.RenderTrustedMarkdown(input))

	// Sanitize HTML, but allow fenced code blocks to not get mangled in user
	// submitted comments.
	p := bluemonday.UGCPolicy()
	p.AllowAttrs("class").Matching(reFencedCodeClass).OnElements("code")
	html := p.SanitizeBytes(unsafe)
	return string(html)
}

// RenderTrustedMarkdown renders markdown to HTML, but without applying
// bluemonday filtering afterward. This is for blog posts and website
// Markdown pages, not for user-submitted comments or things.
func (b *Blog) RenderTrustedMarkdown(input string) string {
	html := github_flavored_markdown.Markdown([]byte(input))
	return string(html)
}
