package telegram

import (
	"github.com/mtgo-labs/mtgo/telegram/types"
	"github.com/mtgo-labs/mtgo/tg"
)

func (c *Context) AnswerInlineQuery(results []tg.InputBotInlineResultClass, opts ...*AnswerInlineQueryOption) error {
	if c.InlineQuery == nil {
		return nil
	}
	return c.Client.AnswerInlineQuery(c.Ctx, c.InlineQuery.ID, results, opts...)
}

func (c *Context) AnswerInline(results []tg.InputBotInlineResultClass, opts ...*AnswerInlineQueryOption) error {
	return c.AnswerInlineQuery(results, opts...)
}

func (c *Context) AnswerInlineResults(results []types.InlineResultBuilder, opts ...*AnswerInlineQueryOption) error {
	if c.InlineQuery == nil {
		return nil
	}
	tlResults := make([]tg.InputBotInlineResultClass, len(results))
	for i, r := range results {
		tlResults[i] = r.TL()
	}
	return c.Client.AnswerInlineQuery(c.Ctx, c.InlineQuery.ID, tlResults, opts...)
}

func (c *Context) AnswerInlineArticle(id, title, text string, opts ...*AnswerInlineQueryOption) error {
	return c.AnswerInlineResults([]types.InlineResultBuilder{
		&types.InlineArticle{ID: id, Title: title, Text: text},
	}, opts...)
}

func (c *Context) AnswerInlineArticles(articles []*types.InlineArticle, opts ...*AnswerInlineQueryOption) error {
	results := make([]types.InlineResultBuilder, len(articles))
	for i, a := range articles {
		results[i] = a
	}
	return c.AnswerInlineResults(results, opts...)
}

func (c *Context) AnswerInlinePhoto(id string, photoID, accessHash int64, text string, opts ...*AnswerInlineQueryOption) error {
	return c.AnswerInlineResults([]types.InlineResultBuilder{
		&types.InlinePhoto{ID: id, PhotoID: photoID, AccessHash: accessHash, Text: text},
	}, opts...)
}

func (c *Context) AnswerInlinePhotos(photos []*types.InlinePhoto, opts ...*AnswerInlineQueryOption) error {
	results := make([]types.InlineResultBuilder, len(photos))
	for i, p := range photos {
		results[i] = p
	}
	return c.AnswerInlineResults(results, opts...)
}

func (c *Context) AnswerInlineDocument(id string, docID, accessHash int64, text string, opts ...*AnswerInlineQueryOption) error {
	return c.AnswerInlineResults([]types.InlineResultBuilder{
		&types.InlineDocument{ID: id, DocumentID: docID, AccessHash: accessHash, Text: text},
	}, opts...)
}

func (c *Context) AnswerInlineDocuments(docs []*types.InlineDocument, opts ...*AnswerInlineQueryOption) error {
	results := make([]types.InlineResultBuilder, len(docs))
	for i, d := range docs {
		results[i] = d
	}
	return c.AnswerInlineResults(results, opts...)
}

func (c *Context) AnswerInlineGame(id, shortName string, opts ...*AnswerInlineQueryOption) error {
	return c.AnswerInlineResults([]types.InlineResultBuilder{
		&types.InlineGame{ID: id, ShortName: shortName},
	}, opts...)
}

func (c *Context) AnswerShipping(queryID int64, ok bool, options []*tg.ShippingOption) error {
	if c.Client == nil {
		return ErrContextNoClient
	}
	return c.Client.AnswerShippingQuery(c.Ctx, queryID, ok, options)
}

func (c *Context) AnswerPreCheckout(queryID int64, ok bool, errorMessage string) error {
	if c.Client == nil {
		return ErrContextNoClient
	}
	return c.Client.AnswerPreCheckoutQuery(c.Ctx, queryID, ok, errorMessage)
}
