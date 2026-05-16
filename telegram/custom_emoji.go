package telegram

import (
	"context"
	"fmt"
	"strconv"

	"github.com/mtgo-labs/mtgo/telegram/types"
	"github.com/mtgo-labs/mtgo/tg"
)

// GetCustomEmojiStickers returns sticker metadata for the given custom emoji
// document identifiers. Telegram accepts at most 200 identifiers per request.
func (c *Client) GetCustomEmojiStickers(ctx context.Context, customEmojiIDs []string) ([]*types.Sticker, error) {
	c.Log.Debugf("GetCustomEmojiStickers count=%d", len(customEmojiIDs))
	if len(customEmojiIDs) == 0 {
		return []*types.Sticker{}, nil
	}
	if len(customEmojiIDs) > 200 {
		return nil, fmt.Errorf("custom emoji IDs exceed limit: got %d, max 200", len(customEmojiIDs))
	}

	documentIDs := make([]int64, 0, len(customEmojiIDs))
	for _, id := range customEmojiIDs {
		documentID, err := strconv.ParseInt(id, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("parse custom emoji ID %q: %w", id, err)
		}
		documentIDs = append(documentIDs, documentID)
	}

	raw, err := c.Raw().Invoke(ctx, &tg.MessagesGetCustomEmojiDocumentsRequest{
		DocumentID: documentIDs,
	}, tg.ReadTLObject)
	if err != nil {
		return nil, fmt.Errorf("get custom emoji stickers: %w", err)
	}

	vector, ok := raw.(*tg.GenericVector)
	if !ok {
		return nil, fmt.Errorf("get custom emoji stickers: unexpected result type %T", raw)
	}

	stickers := make([]*types.Sticker, 0, len(vector.Items))
	for _, item := range vector.Items {
		doc, ok := item.(*tg.Document)
		if !ok {
			return nil, fmt.Errorf("get custom emoji stickers: unexpected document type %T", item)
		}
		if sticker := types.ParseSticker(doc); sticker != nil {
			stickers = append(stickers, sticker)
		}
	}
	return stickers, nil
}
