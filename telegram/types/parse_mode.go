package types

import "github.com/mtgo-labs/mtgo/telegram/params"

// ParseMode is a type alias for the params.ParseMode type, representing the
// text formatting mode used in message captions and text content.
type ParseMode = params.ParseMode

const (
	// ParseModeDefault uses Telegram's default text formatting (no parsing).
	ParseModeDefault = params.ParseModeDefault
	// ParseModeMarkdown parses text as Markdown formatting.
	ParseModeMarkdown = params.ParseModeMarkdown
	// ParseModeHTML parses text as HTML formatting.
	ParseModeHTML = params.ParseModeHTML
	// ParseModeDisabled disables all text parsing, sending raw text.
	ParseModeDisabled = params.ParseModeDisabled

	// Markdown is a shorthand for ParseModeMarkdown.
	Markdown = params.Markdown
	// HTML is a shorthand for ParseModeHTML.
	HTML = params.HTML
	// MarkdownV2 parses text as MarkdownV2 formatting.
	MarkdownV2 = params.MarkdownV2
	// Disabled is a shorthand for ParseModeDisabled.
	Disabled = params.Disabled
)
