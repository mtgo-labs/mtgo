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

func (c *Client) BoundStoryReplyText(peerID int64, storyID int32, text string, opts ...*params.SendMessage) (*types.Message, error) {
	return c.SendMessage(context.Background(), peerID, text, storyReplyOpt(peerID, storyID))
}

func (c *Client) BoundStoryReplyAnimation(peerID int64, storyID int32, media tg.InputMediaClass, caption string, opts ...*params.SendMessage) (*types.Message, error) {
	return c.SendMedia(context.Background(), peerID, media, caption, storyReplyOpt(peerID, storyID))
}

func (c *Client) BoundStoryReplyAudio(peerID int64, storyID int32, media tg.InputMediaClass, caption string, opts ...*params.SendMessage) (*types.Message, error) {
	return c.SendMedia(context.Background(), peerID, media, caption, storyReplyOpt(peerID, storyID))
}

func (c *Client) BoundStoryReplyCachedMedia(peerID int64, storyID int32, fileID string, opts ...*params.SendMessage) (*types.Message, error) {
	return c.SendCachedMedia(context.Background(), peerID, fileID, storyReplyOpt(peerID, storyID))
}

func (c *Client) BoundStoryReplyMediaGroup(peerID int64, storyID int32, media []tg.InputMediaClass, opts ...*params.SendMessage) ([]*types.Message, error) {
	opt := storyReplyOpt(peerID, storyID)
	items := make([]*tg.InputSingleMedia, len(media))
	for i, m := range media {
		items[i] = &tg.InputSingleMedia{Media: m, RandomID: c.RandomID()}
	}
	return c.SendMediaGroup(context.Background(), peerID, items, opt)
}

func (c *Client) BoundStoryReplyPhoto(peerID int64, storyID int32, media tg.InputMediaClass, caption string, opts ...*params.SendMessage) (*types.Message, error) {
	return c.SendMedia(context.Background(), peerID, media, caption, storyReplyOpt(peerID, storyID))
}

func (c *Client) BoundStoryReplySticker(peerID int64, storyID int32, media tg.InputMediaClass, opts ...*params.SendMessage) (*types.Message, error) {
	return c.SendMedia(context.Background(), peerID, media, "", storyReplyOpt(peerID, storyID))
}

func (c *Client) BoundStoryReplyVideo(peerID int64, storyID int32, media tg.InputMediaClass, caption string, opts ...*params.SendMessage) (*types.Message, error) {
	return c.SendMedia(context.Background(), peerID, media, caption, storyReplyOpt(peerID, storyID))
}

func (c *Client) BoundStoryReplyVideoNote(peerID int64, storyID int32, media tg.InputMediaClass, opts ...*params.SendMessage) (*types.Message, error) {
	return c.SendMedia(context.Background(), peerID, media, "", storyReplyOpt(peerID, storyID))
}

func (c *Client) BoundStoryReplyVoice(peerID int64, storyID int32, media tg.InputMediaClass, caption string, opts ...*params.SendMessage) (*types.Message, error) {
	return c.SendMedia(context.Background(), peerID, media, caption, storyReplyOpt(peerID, storyID))
}

func (c *Client) BoundStoryCopy(peerID int64, storyID int32, toChatID int64) (*types.Message, error) {
	return c.BoundStoryForward(peerID, storyID, toChatID)
}

func (c *Client) BoundStoryView(peerID int64, storyID int32) error {
	return c.BoundStoryRead(peerID, storyID)
}
