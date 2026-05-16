package types

import (
	"github.com/mtgo-labs/mtgo/tg"
)

// InlineArticle builds an inline query result of type "article" for sending
// text content with an optional thumbnail.
//
// Example:
//
//	article := &types.InlineArticle{ID: "1", Title: "Hello", Text: "World"}
//	result := article.TL()
type InlineArticle struct {
	ID          string
	Title       string
	Description string
	URL         string
	ThumbURL    string
	ThumbWidth  int32
	ThumbHeight int32
	HideURL     bool
	Text        string
	ParseMode   ParseMode
}

func (a *InlineArticle) TL() tg.InputBotInlineResultClass {
	var flags tg.Fields
	if a.Title != "" {
		flags.Set(1)
	}
	if a.Description != "" {
		flags.Set(2)
	}
	if a.URL != "" {
		flags.Set(3)
	}
	var thumb *tg.InputWebDocument
	if a.ThumbURL != "" {
		flags.Set(4)
		thumb = &tg.InputWebDocument{
			URL:      a.ThumbURL,
			MimeType: "image/jpeg",
		}
	}
	msgFlags := parseModeFlags(a.ParseMode)
	return &tg.InputBotInlineResult{
		Flags:       flags,
		ID:          a.ID,
		Type:        "article",
		Title:       a.Title,
		Description: a.Description,
		URL:         a.URL,
		Thumb:       thumb,
		SendMessage: &tg.InputBotInlineMessageText{
			Flags:   msgFlags,
			Message: a.Text,
		},
	}
}

// InlinePhoto builds an inline query result referencing an existing photo by ID.
//
// Example:
//
//	photo := &types.InlinePhoto{ID: "2", PhotoID: docID, AccessHash: hash, Text: "caption"}
//	result := photo.TL()
type InlinePhoto struct {
	ID          string
	Title       string
	Description string
	PhotoID     int64
	AccessHash  int64
	FileRef     []byte
	Text        string
	ParseMode   ParseMode
}

func (p *InlinePhoto) TL() tg.InputBotInlineResultClass {
	return &tg.InputBotInlineResultPhoto{
		ID:   p.ID,
		Type: "photo",
		Photo: &tg.InputPhoto{
			ID:            p.PhotoID,
			AccessHash:    p.AccessHash,
			FileReference: p.FileRef,
		},
		SendMessage: &tg.InputBotInlineMessageMediaAuto{
			Flags:   parseModeFlags(p.ParseMode),
			Message: p.Text,
		},
	}
}

// InlineDocument builds an inline query result referencing an existing document by ID.
//
// Example:
//
//	doc := &types.InlineDocument{ID: "3", DocumentID: docID, AccessHash: hash, Type: "gif"}
//	result := doc.TL()
type InlineDocument struct {
	ID          string
	Title       string
	Description string
	Type        string
	DocumentID  int64
	AccessHash  int64
	FileRef     []byte
	Text        string
	ParseMode   ParseMode
}

func (d *InlineDocument) TL() tg.InputBotInlineResultClass {
	var flags tg.Fields
	if d.Title != "" {
		flags.Set(1)
	}
	if d.Description != "" {
		flags.Set(2)
	}
	resultType := d.Type
	if resultType == "" {
		resultType = "document"
	}
	return &tg.InputBotInlineResultDocument{
		Flags:       flags,
		ID:          d.ID,
		Type:        resultType,
		Title:       d.Title,
		Description: d.Description,
		Document: &tg.InputDocument{
			ID:            d.DocumentID,
			AccessHash:    d.AccessHash,
			FileReference: d.FileRef,
		},
		SendMessage: &tg.InputBotInlineMessageMediaAuto{
			Flags:   parseModeFlags(d.ParseMode),
			Message: d.Text,
		},
	}
}

// InlineGame builds an inline query result for a game.
//
// Example:
//
//	game := &types.InlineGame{ID: "4", ShortName: "mygame"}
//	result := game.TL()
type InlineGame struct {
	ID        string
	ShortName string
	Text      string
	ParseMode ParseMode
}

func (g *InlineGame) TL() tg.InputBotInlineResultClass {
	return &tg.InputBotInlineResultGame{
		ID:          g.ID,
		ShortName:   g.ShortName,
		SendMessage: &tg.InputBotInlineMessageGame{},
	}
}

func parseModeFlags(pm ParseMode) tg.Fields {
	if pm == ParseModeHTML || pm == ParseModeMarkdown || pm == MarkdownV2 {
		return tg.Fields(1 << 1)
	}
	return 0
}

// InlineResultBuilder is the interface for types that can produce a TL inline
// result for answering inline queries.
type InlineResultBuilder interface {
	TL() tg.InputBotInlineResultClass
}

func buildInlineResults(results []InlineResultBuilder) []tg.InputBotInlineResultClass {
	out := make([]tg.InputBotInlineResultClass, len(results))
	for i, r := range results {
		out[i] = r.TL()
	}
	return out
}
