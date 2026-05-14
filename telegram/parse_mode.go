package telegram

import "github.com/mtgo-labs/mtgo/telegram/params"

// ParseMode is the text formatting mode for messages.
// Use the short constants: Markdown, HTML, MarkdownV2, Disabled.
type ParseMode = params.ParseMode

const (
	// Markdown formats message text as Markdown.
	Markdown = params.Markdown
	// HTML formats message text as HTML.
	HTML = params.HTML
	// MarkdownV2 formats message text as MarkdownV2.
	MarkdownV2 = params.MarkdownV2
	// Disabled sends raw text with no formatting.
	Disabled = params.Disabled
)
