package core

import (
	"bytes"
	"errors"
	"fmt"
	"os/exec"
	"regexp"
	"strings"

	"github.com/microcosm-cc/bluemonday"
	"github.com/shurcooL/github_flavored_markdown"
)

// Regexps for Markdown use cases.
var (
	// Match title from the first `# h1` heading.
	reMarkdownTitle = regexp.MustCompile(`(?m:^#([^#\r\n]+)$)`)

	// Match fenced code blocks with languages defined.
	reFencedCode = regexp.MustCompile("```" + `([a-z]*)\n([\s\S]*?)\n\s*` + "```")

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
	// Find and hang on to fenced code blocks.
	codeBlocks := []codeBlock{}
	log.Info("RE: %s", reFencedCode.String())
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
		highlighted, _ := Pygmentize(block.language, block.source)
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
// TODO: this takes ~0.6s per go, we need something faster.
func Pygmentize(language, source string) (string, error) {
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

	return out.String(), nil
}
