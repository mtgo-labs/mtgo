package telegram

import (
	"context"

	"github.com/mtgo-labs/mtgo/tg"
)

func (c *Client) BoundAddToGifs(ctx context.Context, doc tg.InputDocumentClass) error {
	_, err := c.Raw().MessagesSaveGIF(ctx, &tg.MessagesSaveGIFRequest{
		ID:     doc,
		Unsave: false,
	})
	return err
}

func (c *Client) BoundRemoveFromGifs(ctx context.Context, doc tg.InputDocumentClass) error {
	_, err := c.Raw().MessagesSaveGIF(ctx, &tg.MessagesSaveGIFRequest{
		ID:     doc,
		Unsave: true,
	})
	return err
}

func (c *Client) BoundGetSavedGifs(ctx context.Context, hash int64) (tg.SavedGifsClass, error) {
	return c.Raw().MessagesGetSavedGifs(ctx, &tg.MessagesGetSavedGifsRequest{
		Hash: hash,
	})
}

func (c *Client) BoundAddToStickers(ctx context.Context, doc tg.InputDocumentClass) error {
	_, err := c.Raw().MessagesFaveSticker(ctx, &tg.MessagesFaveStickerRequest{
		ID:     doc,
		Unfave: false,
	})
	return err
}

func (c *Client) BoundRemoveFromStickers(ctx context.Context, doc tg.InputDocumentClass) error {
	_, err := c.Raw().MessagesFaveSticker(ctx, &tg.MessagesFaveStickerRequest{
		ID:     doc,
		Unfave: true,
	})
	return err
}
