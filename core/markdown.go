package core

import "github.com/shurcooL/github_flavored_markdown"

// RenderMarkdown renders markdown to HTML.
func (b *Blog) RenderMarkdown(input string) string {
	output := github_flavored_markdown.Markdown([]byte(input))
	return string(output)
}
