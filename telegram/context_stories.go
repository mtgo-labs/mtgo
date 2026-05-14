package telegram

import (
	"fmt"

	"github.com/mtgo-labs/mtgo/telegram/types"
	"github.com/mtgo-labs/mtgo/tg"
)

// SendStory posts a new story on behalf of the specified chat. The chat must support
// stories (typically channels or the user's own chat).
//
// Parameters:
//   - chatID: the chat ID to post the story on
//   - media: the media content for the story as an [tg.InputMediaClass]
//   - opts: optional [SendStoryOption] parameters for caption, privacy, and other settings
//
// Returns:
//   - *types.Story: the created story
//   - error: non-nil if the context has no client, the chat cannot post stories, or the upload fails
func (c *Context) SendStory(chatID int64, media tg.InputMediaClass, opts ...*SendStoryOption) (*types.Story, error) {
	if c.Client == nil {
		return nil, fmt.Errorf("context: no client")
	}
	return c.Client.SendStory(c.Ctx, chatID, media, opts...)
}

// EditStoryCaption changes the caption text of an existing story.
//
// Parameters:
//   - chatID: the chat ID that owns the story
//   - storyID: the story ID to edit
//   - caption: the new caption text
//
// Returns:
//   - *types.Story: the updated story
//   - error: non-nil if the context has no client, the story does not exist, or the edit fails
func (c *Context) EditStoryCaption(chatID int64, storyID int32, caption string) (*types.Story, error) {
	if c.Client == nil {
		return nil, fmt.Errorf("context: no client")
	}
	return c.Client.EditStoryCaption(c.Ctx, chatID, storyID, caption)
}

// EditStoryMedia replaces the media content of an existing story.
//
// Parameters:
//   - chatID: the chat ID that owns the story
//   - storyID: the story ID to edit
//   - media: the new media content as an [tg.InputMediaClass]
//
// Returns:
//   - *types.Story: the updated story
//   - error: non-nil if the context has no client, the story does not exist, or the upload fails
func (c *Context) EditStoryMedia(chatID int64, storyID int32, media tg.InputMediaClass) (*types.Story, error) {
	if c.Client == nil {
		return nil, fmt.Errorf("context: no client")
	}
	return c.Client.EditStoryMedia(c.Ctx, chatID, storyID, media)
}

// DeleteStories removes one or more stories from the specified chat.
//
// Parameters:
//   - chatID: the chat ID that owns the stories
//   - storyIDs: slice of story IDs to delete
//
// Returns:
//   - error: non-nil if the context has no client or the deletion fails
func (c *Context) DeleteStories(chatID int64, storyIDs []int32) error {
	if c.Client == nil {
		return fmt.Errorf("context: no client")
	}
	return c.Client.DeleteStories(c.Ctx, chatID, storyIDs)
}

// GetStories retrieves one or more stories by their IDs from the specified user.
//
// Parameters:
//   - userID: the Telegram user ID who posted the stories
//   - storyIDs: slice of story IDs to retrieve
//
// Returns:
//   - []*types.Story: the requested stories
//   - error: non-nil if the context has no client, the stories cannot be found, or the request fails
func (c *Context) GetStories(userID int64, storyIDs []int32) ([]*types.Story, error) {
	if c.Client == nil {
		return nil, fmt.Errorf("context: no client")
	}
	return c.Client.GetStories(c.Ctx, userID, storyIDs)
}

// GetChatStories retrieves all active stories posted by the specified chat.
//
// Parameters:
//   - chatID: the chat ID to retrieve stories from
//
// Returns:
//   - []*types.Story: the chat's active stories
//   - error: non-nil if the context has no client or the request fails
func (c *Context) GetChatStories(chatID int64) ([]*types.Story, error) {
	if c.Client == nil {
		return nil, fmt.Errorf("context: no client")
	}
	return c.Client.GetChatStories(c.Ctx, chatID)
}

// GetStoryViews retrieves the view count and viewer information for the specified stories.
//
// Parameters:
//   - chatID: the chat ID that owns the stories
//   - storyIDs: slice of story IDs to retrieve views for
//
// Returns:
//   	- []*tg.StoryViews: view information for each requested story
//   - error: non-nil if the context has no client or the request fails
func (c *Context) GetStoryViews(chatID int64, storyIDs []int32) ([]*tg.StoryViews, error) {
	if c.Client == nil {
		return nil, fmt.Errorf("context: no client")
	}
	return c.Client.GetStoryViews(c.Ctx, chatID, storyIDs)
}

// ForwardStory forwards a story from one chat to another as a message.
//
// Parameters:
//   - target: the target chat ID to forward the story to
//   - source: the source chat ID that owns the story
//   - storyID: the story ID to forward
//
// Returns:
//   - *types.Message: the forwarded message containing the story
//   - error: non-nil if the context has no client, the story cannot be found, or the forward fails
func (c *Context) ForwardStory(target int64, source int64, storyID int32) (*types.Message, error) {
	if c.Client == nil {
		return nil, fmt.Errorf("context: no client")
	}
	return c.Client.ForwardStory(c.Ctx, target, source, storyID)
}

// ReadChatStories marks the specified stories as read in the chat.
//
// Parameters:
//   - chatID: the chat ID that owns the stories
//   - storyIDs: slice of story IDs to mark as read
//
// Returns:
//   - error: non-nil if the context has no client or the operation fails
func (c *Context) ReadChatStories(chatID int64, storyIDs []int32) error {
	if c.Client == nil {
		return fmt.Errorf("context: no client")
	}
	return c.Client.ReadChatStories(c.Ctx, chatID, storyIDs)
}
