// Package parser converts Telegram HTML and Markdown markup into plain text
// accompanied by MessageEntity slices suitable for the MTProto API.
//
// Two parsers are provided:
//
//   - HTMLParser  — handles <b>, <i>, <u>, <s>, <code>, <pre>, <a href>, <tg-spoiler>,
//     <blockquote>, and custom <emoji> tags.
//   - MarkdownParser — handles **bold**, __italic__, ~~strike~~, `code`, ```pre```,
//     ||spoiler||, and [text](url) links.
//
// Both parsers normalise Unicode and validate entity boundaries against UTF-8
// rune positions required by the Telegram API.
package parser

import (
	"fmt"

	tl "github.com/mtgo-labs/mtgo/tg"
)

// ParseMode represents the text formatting mode used to parse message content
// into Telegram-compatible entities.
type ParseMode int

const (
	// ParseModeDefault performs no parsing and returns the raw text with no entities.
	ParseModeDefault ParseMode = iota
	// ParseModeHTML interprets the input as Telegram HTML markup.
	ParseModeHTML
	// ParseModeMarkdown interprets the input as Telegram Markdown formatting.
	ParseModeMarkdown
	// ParseModeDisabled skips all parsing, returning the text unchanged.
	ParseModeDisabled
)

var (
	htmlParser     HTMLParser
	markdownParser MarkdownParser
)

// Parse parses text according to the given ParseMode and returns the plain text
// alongside the resulting Telegram message entities.
// It returns an error if the mode is not recognized.
func Parse(mode ParseMode, text string) (string, []tl.MessageEntityClass, error) {
	switch mode {
	case ParseModeHTML:
		return htmlParser.Parse(text)
	case ParseModeMarkdown:
		return markdownParser.Parse(text)
	case ParseModeDisabled, ParseModeDefault:
		return text, nil, nil
	default:
		return "", nil, fmt.Errorf("unsupported parse mode: %d", mode)
	}
}
