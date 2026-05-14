package telegram

import (
	"context"

	"github.com/mtgo-labs/mtgo/tg"
)

func (c *Client) BoundAddToGifs(doc tg.InputDocumentClass) error {
	ctx := context.Background()
	_, err := c.Raw().MessagesSaveGIF(ctx, &tg.MessagesSaveGIFRequest{
		ID:     doc,
		Unsave: false,
	})
	return err
}

func (c *Client) BoundRemoveFromGifs(doc tg.InputDocumentClass) error {
	ctx := context.Background()
	_, err := c.Raw().MessagesSaveGIF(ctx, &tg.MessagesSaveGIFRequest{
		ID:     doc,
		Unsave: true,
	})
	return err
}

func (c *Client) BoundGetSavedGifs(hash int64) (tg.SavedGifsClass, error) {
	ctx := context.Background()
	return c.Raw().MessagesGetSavedGifs(ctx, &tg.MessagesGetSavedGifsRequest{
		Hash: hash,
	})
}

func (c *Client) BoundAddToStickers(doc tg.InputDocumentClass) error {
	ctx := context.Background()
	_, err := c.Raw().MessagesFaveSticker(ctx, &tg.MessagesFaveStickerRequest{
		ID:     doc,
		Unfave: false,
	})
	return err
}

func (c *Client) BoundRemoveFromStickers(doc tg.InputDocumentClass) error {
	ctx := context.Background()
	_, err := c.Raw().MessagesFaveSticker(ctx, &tg.MessagesFaveStickerRequest{
		ID:     doc,
		Unfave: true,
	})
	return err
}
