package telegram

import (
	"context"

	"github.com/mtgo-labs/mtgo/telegram/params"
	"github.com/mtgo-labs/mtgo/telegram/types"
	"github.com/mtgo-labs/mtgo/tg"
)

func storyReplyOpt(peerID int64, storyID int32) *params.SendMessage {
	return &params.SendMessage{
		ReplyTo: &tg.InputReplyToStory{
			Peer:    &tg.InputPeerUser{UserID: peerID},
			StoryID: storyID,
		},
	}
}

func (c *Client) BoundStoryReplyText(ctx context.Context, peerID int64, storyID int32, text string, opts ...*params.SendMessage) (*types.Message, error) {
	return c.SendMessage(ctx, peerID, text, storyReplyOpt(peerID, storyID))
}

func (c *Client) BoundStoryReplyAnimation(ctx context.Context, peerID int64, storyID int32, media tg.InputMediaClass, caption string, opts ...*params.SendMessage) (*types.Message, error) {
	return c.SendMedia(ctx, peerID, media, caption, storyReplyOpt(peerID, storyID))
}

func (c *Client) BoundStoryReplyAudio(ctx context.Context, peerID int64, storyID int32, media tg.InputMediaClass, caption string, opts ...*params.SendMessage) (*types.Message, error) {
	return c.SendMedia(ctx, peerID, media, caption, storyReplyOpt(peerID, storyID))
}

func (c *Client) BoundStoryReplyCachedMedia(ctx context.Context, peerID int64, storyID int32, fileID string, opts ...*params.SendMessage) (*types.Message, error) {
	return c.SendCachedMedia(ctx, peerID, fileID, storyReplyOpt(peerID, storyID))
}

func (c *Client) BoundStoryReplyMediaGroup(ctx context.Context, peerID int64, storyID int32, media []tg.InputMediaClass, opts ...*params.SendMessage) ([]*types.Message, error) {
	opt := storyReplyOpt(peerID, storyID)
	items := make([]*tg.InputSingleMedia, len(media))
	for i, m := range media {
		items[i] = &tg.InputSingleMedia{Media: m, RandomID: c.RandomID()}
	}
	return c.SendMediaGroup(ctx, peerID, items, opt)
}

func (c *Client) BoundStoryReplyPhoto(ctx context.Context, peerID int64, storyID int32, media tg.InputMediaClass, caption string, opts ...*params.SendMessage) (*types.Message, error) {
	return c.SendMedia(ctx, peerID, media, caption, storyReplyOpt(peerID, storyID))
}

func (c *Client) BoundStoryReplySticker(ctx context.Context, peerID int64, storyID int32, media tg.InputMediaClass, opts ...*params.SendMessage) (*types.Message, error) {
	return c.SendMedia(ctx, peerID, media, "", storyReplyOpt(peerID, storyID))
}

func (c *Client) BoundStoryReplyVideo(ctx context.Context, peerID int64, storyID int32, media tg.InputMediaClass, caption string, opts ...*params.SendMessage) (*types.Message, error) {
	return c.SendMedia(ctx, peerID, media, caption, storyReplyOpt(peerID, storyID))
}

func (c *Client) BoundStoryReplyVideoNote(ctx context.Context, peerID int64, storyID int32, media tg.InputMediaClass, opts ...*params.SendMessage) (*types.Message, error) {
	return c.SendMedia(ctx, peerID, media, "", storyReplyOpt(peerID, storyID))
}

func (c *Client) BoundStoryReplyVoice(ctx context.Context, peerID int64, storyID int32, media tg.InputMediaClass, caption string, opts ...*params.SendMessage) (*types.Message, error) {
	return c.SendMedia(ctx, peerID, media, caption, storyReplyOpt(peerID, storyID))
}

func (c *Client) BoundStoryCopy(ctx context.Context, peerID int64, storyID int32, toChatID int64) (*types.Message, error) {
	return c.BoundStoryForward(ctx, peerID, storyID, toChatID)
}

func (c *Client) BoundStoryView(ctx context.Context, peerID int64, storyID int32) error {
	return c.BoundStoryRead(ctx, peerID, storyID)
}
