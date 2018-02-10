// Package markdown implements a GitHub Flavored Markdown renderer.
package markdown

import (
	"bytes"
	"crypto/md5"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"regexp"
	"strings"

	"github.com/kirsle/blog/core/internal/log"
	"github.com/kirsle/blog/jsondb/caches"
	"github.com/microcosm-cc/bluemonday"
	"github.com/shurcooL/github_flavored_markdown"
)

// Regexps for Markdown use cases.
var (
	// Plug your own Redis cacher in.
	Cache caches.Cacher

	// Match title from the first `# h1` heading.
	reMarkdownTitle = regexp.MustCompile(`(?m:^#([^#\r\n]+)$)`)

	// Match fenced code blocks with languages defined.
	reFencedCode      = regexp.MustCompile("```" + `([a-z]*)[\r\n]([\s\S]*?)[\r\n]\s*` + "```")
	reFencedCodeClass = regexp.MustCompile("^highlight highlight-[a-zA-Z0-9]+$")

	// Regexp to match fenced code blocks in rendered Markdown HTML.
	// Tweak this if you change Markdown engines later.
	reCodeBlock   = regexp.MustCompile(`<div class="highlight highlight-(.+?)"><pre>(.+?)</pre></div>`)
	reDecodeBlock = regexp.MustCompile(`\[?FENCED_CODE_%d_BLOCK?\]`)
)

// A container for parsed code blocks.
type codeBlock struct {
	placeholder int
	language    string
	source      string
}

// TitleFromMarkdown tries to find a title from the source of a Markdown file.
//
// On error, returns "Untitled" along with the error. So if you're lazy and
// want a suitable default, you can safely ignore the error.
func TitleFromMarkdown(body string) (string, error) {
	m := reMarkdownTitle.FindStringSubmatch(body)
	if len(m) > 0 {
		return m[1], nil
	}
	return "Untitled", errors.New(
		"did not find a single h1 (denoted by # prefix) for Markdown title",
	)
}

// RenderMarkdown renders markdown to HTML, safely. It uses blackfriday to
// render Markdown to HTML and then Bluemonday to sanitize the resulting HTML.
func RenderMarkdown(input string) string {
	unsafe := []byte(RenderTrustedMarkdown(input))

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
func RenderTrustedMarkdown(input string) string {
	// Find and hang on to fenced code blocks.
	codeBlocks := []codeBlock{}
	matches := reFencedCode.FindAllStringSubmatch(input, -1)
	for i, m := range matches {
		language, source := m[1], m[2]
		if language == "" {
			continue
		}
		codeBlocks = append(codeBlocks, codeBlock{i, language, source})

		input = strings.Replace(input, m[0], fmt.Sprintf(
			"[?FENCED_CODE_%d_BLOCK?]",
			i,
		), 1)
	}

	// Render the HTML out.
	html := string(github_flavored_markdown.Markdown([]byte(input)))

	// Substitute fenced codes back in.
	for _, block := range codeBlocks {
		highlighted, err := Pygmentize(block.language, block.source)
		if err != nil {
			log.Error("Pygmentize error: %s", err)
		}
		html = strings.Replace(html,
			fmt.Sprintf("[?FENCED_CODE_%d_BLOCK?]", block.placeholder),
			highlighted,
			1,
		)
	}

	return string(html)
}

// Pygmentize searches for fenced code blocks in rendered Markdown HTML
// and runs Pygments to syntax highlight it.
//
// On error the original given source is returned back.
//
// The rendered result is cached in Redis if available, because the CLI
// call takes ~0.6s which is slow if you're rendering a lot of code blocks.
func Pygmentize(language, source string) (string, error) {
	var result string

	// Hash the source for the cache key.
	h := md5.New()
	io.WriteString(h, language+source)
	hash := fmt.Sprintf("%x", h.Sum(nil))
	cacheKey := "pygmentize:" + hash

	// Do we have it cached?
	if cached, err := Cache.Get(cacheKey); err == nil && len(cached) > 0 {
		return string(cached), nil
	}

	// Defer to the `pygmentize` command
	bin := "pygmentize"
	if _, err := exec.LookPath(bin); err != nil {
		return source, errors.New("pygmentize not installed")
	}

	cmd := exec.Command(bin, "-l"+language, "-f"+"html", "-O encoding=utf-8")
	cmd.Stdin = strings.NewReader(source)

	var out bytes.Buffer
	cmd.Stdout = &out

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		log.Error("Error running pygments: %s", stderr.String())
		return source, err
	}

	result = out.String()
	err := Cache.Set(cacheKey, []byte(result), 60*60*24) // cool md5's don't change
	if err != nil {
		log.Error("Couldn't cache Pygmentize output: %s", err)
	}

	return result, nil
}
