package parser

import (
	"strings"

	tl "github.com/mtgo-labs/mtgo/tg"
)

// MarkdownParser parses Telegram Markdown-formatted text into plain text and message entities.
type MarkdownParser struct{}

// NewMarkdownParser returns a new MarkdownParser ready for use.
func NewMarkdownParser() *MarkdownParser {
	return &MarkdownParser{}
}

// Parse converts Markdown-formatted text to HTML and delegates to HTMLParser.
// It returns the plain text and corresponding Telegram message entities.
func (p *MarkdownParser) Parse(md string) (string, []tl.MessageEntityClass, error) {
	html := mdToHTML(md)
	return htmlParser.Parse(html)
}

func mdToHTML(md string) string {
	s := md

	s = replaceDelimited(s, "```", "<pre>", "</pre>")
	s = replaceDelimited(s, "**", "<b>", "</b>")
	s = replaceDelimited(s, "__", "<b>", "</b>")
	s = replaceDelimited(s, "~~", "<s>", "</s>")
	s = replaceDelimited(s, "||", "<spoiler>", "</spoiler>")
	s = replaceDelimited(s, "`", "<code>", "</code>")
	s = replaceDelimited(s, "*", "<i>", "</i>")
	s = replaceDelimited(s, "_", "<i>", "</i>")

	return s
}

func replaceDelimited(s, delim, openTag, closeTag string) string {
	var b strings.Builder
	b.Grow(len(s))
	open := true
	for {
		idx := strings.Index(s, delim)
		if idx == -1 {
			b.WriteString(s)
			break
		}
		b.WriteString(s[:idx])
		if open {
			b.WriteString(openTag)
		} else {
			b.WriteString(closeTag)
		}
		s = s[idx+len(delim):]
		open = !open
	}
	return b.String()
}
