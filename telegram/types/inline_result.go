package types

import "github.com/mtgo-labs/mtgo/tg"

// InlineQueryResult represents a generic inline query result returned to the user.
// Displayed in the inline mode suggestion list when the user types a query to a bot.
type InlineQueryResult struct {
	// Type is the result type identifier (e.g. "photo", "article", "gif").
	Type string
	// ID is the unique identifier for this inline result within the query session.
	ID string
	// Title is the display title of the result shown to the user.
	Title string
	// Description is a short text description shown below the title.
	Description string
	// URL is the URL of the result, opened when the result is selected.
	URL string
	// ThumbURL is the URL of the thumbnail image for the result.
	ThumbURL string
	// ThumbWidth is the width of the thumbnail in pixels.
	ThumbWidth int32
	// ThumbHeight is the height of the thumbnail in pixels.
	ThumbHeight int32
}

// ParseInlineQueryResult converts a TL InputBotInlineResultClass into an InlineQueryResult.
// Returns nil if raw is nil.
func ParseInlineQueryResult(raw tg.InputBotInlineResultClass) *InlineQueryResult {
	if raw == nil {
		return nil
	}
	r := &InlineQueryResult{}
	switch v := raw.(type) {
	case *tg.InputBotInlineResult:
		r.ID = v.ID
		r.Type = v.Type
		if v.Title != "" {
			r.Title = v.Title
		}
		if v.Description != "" {
			r.Description = v.Description
		}
		if v.URL != "" {
			r.URL = v.URL
		}
		if v.Thumb != nil {
			r.ThumbURL = v.Thumb.URL
		}
		if v.Content != nil {
			r.URL = v.Content.URL
		}
	case *tg.InputBotInlineResultPhoto:
		r.ID = v.ID
		r.Type = v.Type
	case *tg.InputBotInlineResultDocument:
		r.ID = v.ID
		r.Type = v.Type
		if v.Title != "" {
			r.Title = v.Title
		}
		if v.Description != "" {
			r.Description = v.Description
		}
	case *tg.InputBotInlineResultGame:
		r.ID = v.ID
		r.Type = "game"
	}
	return r
}
