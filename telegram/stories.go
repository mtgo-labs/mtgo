package telegram

import (
	"context"
	"fmt"

	"github.com/mtgo-labs/mtgo/telegram/types"
	"github.com/mtgo-labs/mtgo/tg"
)

// SendStoryOption configures optional parameters when posting a new story using
// [Client.SendStory].
type SendStoryOption struct {
	// Pinned pins the story to the top of the profile immediately after posting.
	Pinned bool
	// NoForwards disables forwarding of the story content.
	NoForwards bool
	// Period is the story expiration period in seconds. nil uses the default
	// (24 hours).
	Period *int32
	// PrivacyRules specifies who can view the story. If empty, the default
	// privacy settings are used.
	PrivacyRules []tg.InputPrivacyRuleClass
}

// SendStory posts a new story to the specified peer's story feed. The media
// parameter must be non-nil and describes the photo or video to post.
//
// Parameters:
//   - ctx: context for cancellation and timeout
//   - chatID: identifier of the peer (user or channel) posting the story
//   - media: media content for the story (photo, video, etc.)
//   - opts: optional [SendStoryOption] configuration
//
// Returns the posted Story object or an error if the media is nil, the peer
// cannot be resolved, or the Telegram API returns an error.
//
// Example:
//
//	ctx := context.Background()
//	media := &tg.InputMediaUploadedPhoto{File: uploadedFile}
//	story, err := client.SendStory(ctx, chatID, media)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	fmt.Printf("Story posted with ID %d\n", story.ID)
func (c *Client) SendStory(ctx context.Context, chatID int64, media tg.InputMediaClass, opts ...*SendStoryOption) (*types.Story, error) {
	c.Log.Debugf("SendStory chat_id=%d", chatID)
	if media == nil {
		return nil, ErrMediaRequired
	}

	peer, err := resolvePeer(c.clientPeerResolver(), chatID)
	if err != nil {
		return nil, fmt.Errorf("resolve peer: %w", err)
	}
	opt := getOptDef(&SendStoryOption{}, opts...)

	var flags tg.Fields
	if opt.Pinned {
		flags.Set(2)
	}
	if opt.NoForwards {
		flags.Set(4)
	}
	if opt.Period != nil {
		flags.Set(8)
	}

	req := &tg.StoriesSendStoryRequest{
		Flags:        flags,
		Pinned:       opt.Pinned,
		Noforwards:   opt.NoForwards,
		Peer:         peer,
		Media:        media,
		RandomID:     c.RandomID(),
		PrivacyRules: opt.PrivacyRules,
	}
	if opt.Period != nil {
		req.Period = *opt.Period
	}

	rpc := c.Raw()
	result, err := rpc.StoriesSendStory(ctx, req)
	if err != nil {
		return nil, err
	}

	return extractStoryFromUpdates(result)
}

// EditStoryCaption updates the caption of an existing story identified by storyID
// owned by chatID. The caption parameter must be non-empty.
//
// Returns the updated Story on success, or an error if the caption is empty,
// the peer cannot be resolved, or the Telegram API returns an error.
//
// Example:
//
//	ctx := context.Background()
//	story, err := client.EditStoryCaption(ctx, chatID, 5, "Updated caption")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	fmt.Printf("Caption updated for story %d\n", story.ID)
func (c *Client) EditStoryCaption(ctx context.Context, chatID int64, storyID int32, caption string) (*types.Story, error) {
	c.Log.Debugf("EditStoryCaption chat_id=%d story_id=%d", chatID, storyID)
	if caption == "" {
		return nil, ErrCaptionRequired
	}

	peer, err := resolvePeer(c.clientPeerResolver(), chatID)
	if err != nil {
		return nil, fmt.Errorf("resolve peer: %w", err)
	}

	var flags tg.Fields
	flags.Set(1)

	req := &tg.StoriesEditStoryRequest{
		Flags:   flags,
		Peer:    peer,
		ID:      storyID,
		Caption: caption,
	}

	rpc := c.Raw()
	result, err := rpc.StoriesEditStory(ctx, req)
	if err != nil {
		return nil, err
	}

	return extractStoryFromUpdates(result)
}

// EditStoryMedia replaces the media content of an existing story identified by
// storyID owned by chatID. The media parameter must be non-nil.
//
// Returns the updated Story on success, or an error if the media is nil,
// the peer cannot be resolved, or the Telegram API returns an error.
func (c *Client) EditStoryMedia(ctx context.Context, chatID int64, storyID int32, media tg.InputMediaClass) (*types.Story, error) {
	c.Log.Debugf("EditStoryMedia chat_id=%d story_id=%d", chatID, storyID)
	if media == nil {
		return nil, ErrMediaRequired
	}

	peer, err := resolvePeer(c.clientPeerResolver(), chatID)
	if err != nil {
		return nil, fmt.Errorf("resolve peer: %w", err)
	}

	var flags tg.Fields
	flags.Set(0)

	req := &tg.StoriesEditStoryRequest{
		Flags: flags,
		Peer:  peer,
		ID:    storyID,
		Media: media,
	}

	rpc := c.Raw()
	result, err := rpc.StoriesEditStory(ctx, req)
	if err != nil {
		return nil, err
	}

	return extractStoryFromUpdates(result)
}

// DeleteStories removes one or more stories from the specified chat.
// The peer must support stories (typically channels or the user's own chat).
//
// Parameters:
//   - ctx: context for cancellation and timeout
//   - chatID: the chat ID that owns the stories
//   - storyIDs: slice of story IDs to delete
//
// Returns an error if the context has no client, no story IDs are provided,
// the peer cannot be resolved, or the Telegram API returns an error.
func (c *Client) DeleteStories(ctx context.Context, chatID int64, storyIDs []int32) error {
	c.Log.Debugf("DeleteStories chat_id=%d story_ids=%v", chatID, storyIDs)
	if len(storyIDs) == 0 {
		return ErrStoryIDsRequired
	}

	peer, err := resolvePeer(c.clientPeerResolver(), chatID)
	if err != nil {
		return fmt.Errorf("resolve peer: %w", err)
	}

	req := &tg.StoriesDeleteStoriesRequest{
		Peer: peer,
		ID:   storyIDs,
	}

	rpc := c.Raw()
	_, err = rpc.StoriesDeleteStories(ctx, req)
	return err
}

// GetStories retrieves one or more stories by ID from the specified user.
// The userID parameter identifies the story owner. The storyIDs parameter
// must contain at least one story ID.
//
// Returns a slice of matching Story objects on success, or an error if no IDs
// are provided, the user cannot be resolved, or the Telegram API returns an error.
//
// Example:
//
//	ctx := context.Background()
//	stories, err := client.GetStories(ctx, userID, []int32{1, 2, 3})
//	if err != nil {
//	    log.Fatal(err)
//	}
//	fmt.Printf("Retrieved %d stories\n", len(stories))
func (c *Client) GetStories(ctx context.Context, userID int64, storyIDs []int32) ([]*types.Story, error) {
	c.Log.Debugf("GetStories user_id=%d count=%d", userID, len(storyIDs))
	if len(storyIDs) == 0 {
		return nil, ErrStoryIDsRequired
	}

	user, err := resolveUserID(c.clientPeerResolver(), userID)
	if err != nil {
		return nil, fmt.Errorf("resolve user: %w", err)
	}

	var peer tg.InputPeerClass
	switch u := user.(type) {
	case *tg.InputUserSelf:
		peer = &tg.InputPeerSelf{}
	case *tg.InputUser:
		peer = &tg.InputPeerUser{UserID: u.UserID, AccessHash: u.AccessHash}
	default:
		return nil, fmt.Errorf("unsupported input user type %T", user)
	}

	req := &tg.StoriesGetStoriesByIDRequest{
		Peer: peer,
		ID:   storyIDs,
	}

	rpc := c.Raw()
	result, err := rpc.StoriesGetStoriesByID(ctx, req)
	if err != nil {
		return nil, err
	}

	stories := make([]*types.Story, 0, len(result.Stories))
	for _, s := range result.Stories {
		if parsed := types.ParseStory(s, nil); parsed != nil {
			stories = append(stories, parsed)
		}
	}
	return stories, nil
}

// GetChatStories retrieves all active stories posted by the specified chat or peer.
//
// Returns a slice of Story objects on success, or an error if the peer cannot be
// resolved, the response has an unexpected type, or the Telegram API returns an error.
//
// Example:
//
//	ctx := context.Background()
//	stories, err := client.GetChatStories(ctx, chatID)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	fmt.Printf("Chat has %d active stories\n", len(stories))
func (c *Client) GetChatStories(ctx context.Context, chatID int64) ([]*types.Story, error) {
	c.Log.Debugf("GetChatStories chat_id=%d", chatID)
	peer, err := resolvePeer(c.clientPeerResolver(), chatID)
	if err != nil {
		return nil, fmt.Errorf("resolve peer: %w", err)
	}

	req := &tg.StoriesGetPeerStoriesRequest{
		Peer: peer,
	}

	rpc := c.Raw()
	result, err := rpc.StoriesGetPeerStories(ctx, req)
	if err != nil {
		return nil, err
	}

	var items []tg.StoryItemClass
	switch v := result.(type) {
	case *tg.PeerStories:
		items = v.Stories
	default:
		return nil, fmt.Errorf("unexpected peer stories type %T", result)
	}

	stories := make([]*types.Story, 0, len(items))
	for _, s := range items {
		if parsed := types.ParseStory(s, nil); parsed != nil {
			stories = append(stories, parsed)
		}
	}
	return stories, nil
}

// GetStoryViews retrieves view statistics for the specified stories owned by chatID.
// The storyIDs parameter must contain at least one story ID.
//
// Returns a slice of StoryViewsTL on success, or an error if no IDs are provided,
// the peer cannot be resolved, the response has an unexpected type, or the
// Telegram API returns an error.
func (c *Client) GetStoryViews(ctx context.Context, chatID int64, storyIDs []int32) ([]*tg.StoryViews, error) {
	c.Log.Debugf("GetStoryViews chat_id=%d count=%d", chatID, len(storyIDs))
	if len(storyIDs) == 0 {
		return nil, ErrStoryIDsRequired
	}

	peer, err := resolvePeer(c.clientPeerResolver(), chatID)
	if err != nil {
		return nil, fmt.Errorf("resolve peer: %w", err)
	}

	req := &tg.StoriesGetStoriesViewsRequest{
		Peer: peer,
		ID:   storyIDs,
	}

	rpc := c.Raw()
	result, err := rpc.StoriesGetStoriesViews(ctx, req)
	if err != nil {
		return nil, err
	}

	var rawViews []tg.StoryViewsClass
	switch v := result.(type) {
	case *tg.StoriesStoryViews:
		rawViews = v.Views
	default:
		return nil, fmt.Errorf("unexpected story views type %T", result)
	}

	views := make([]*tg.StoryViews, 0, len(rawViews))
	for _, v := range rawViews {
		if sv, ok := v.(*tg.StoryViews); ok {
			views = append(views, sv)
		}
	}
	return views, nil
}

// ForwardStory forwards a story from sourceChatID to targetChatID.
// The storyID parameter identifies the story to forward.
//
// Returns the resulting Message on success, or an error if either peer cannot
// be resolved or the Telegram API returns an error.
func (c *Client) ForwardStory(ctx context.Context, targetChatID int64, sourceChatID int64, storyID int32) (*types.Message, error) {
	c.Log.Debugf("ForwardStory to=%d from=%d", targetChatID, sourceChatID)
	targetPeer, err := resolvePeer(c.clientPeerResolver(), targetChatID)
	if err != nil {
		return nil, fmt.Errorf("resolve target peer: %w", err)
	}

	sourcePeer, err := resolvePeer(c.clientPeerResolver(), sourceChatID)
	if err != nil {
		return nil, fmt.Errorf("resolve source peer: %w", err)
	}

	req := &tg.MessagesSendMediaRequest{
		Peer:     targetPeer,
		Media:    &tg.InputMediaStory{Peer: sourcePeer, ID: storyID},
		RandomID: c.RandomID(),
	}

	rpc := c.Raw()
	result, err := rpc.MessagesSendMedia(ctx, req)
	if err != nil {
		return nil, err
	}

	return extractSingleMessage(result, c)
}

// PinChatStories pins the specified stories to the top of the chat's story list.
// The storyIDs parameter specifies which stories to pin.
//
// Returns an error if the peer cannot be resolved or the Telegram API returns an error.
func (c *Client) PinChatStories(ctx context.Context, chatID int64, storyIDs []int32) error {
	peer, err := resolvePeer(c.clientPeerResolver(), chatID)
	if err != nil {
		return fmt.Errorf("resolve peer: %w", err)
	}

	req := &tg.StoriesTogglePinnedToTopRequest{
		Peer: peer,
		ID:   storyIDs,
	}

	rpc := c.Raw()
	_, err = rpc.StoriesTogglePinnedToTop(ctx, req)
	return err
}

// ReadChatStories marks the specified stories as read (increments their view count)
// for the given chat.
//
// Returns an error if the peer cannot be resolved or the Telegram API returns an error.
func (c *Client) ReadChatStories(ctx context.Context, chatID int64, storyIDs []int32) error {
	peer, err := resolvePeer(c.clientPeerResolver(), chatID)
	if err != nil {
		return fmt.Errorf("resolve peer: %w", err)
	}

	req := &tg.StoriesIncrementStoryViewsRequest{
		Peer: peer,
		ID:   storyIDs,
	}

	rpc := c.Raw()
	_, err = rpc.StoriesIncrementStoryViews(ctx, req)
	return err
}

func extractStoryFromUpdates(result tg.UpdatesClass) (*types.Story, error) {
	switch v := result.(type) {
	case *tg.Updates:
		return findStoryInUpdates(v.Updates, v.Users, v.Chats)
	case *tg.UpdatesCombined:
		return findStoryInUpdates(v.Updates, v.Users, v.Chats)
	case *tg.UpdateShort:
		if upd, ok := v.Update.(*tg.UpdateStory); ok {
			return types.ParseStory(upd.Story, nil), nil
		}
		return nil, ErrStoryNotInShort
	default:
		return nil, fmt.Errorf("unexpected updates type %T", result)
	}
}

func findStoryInUpdates(updates []tg.UpdateClass, users []tg.UserClass, chats []tg.ChatClass) (*types.Story, error) {
	pm := types.NewPeerMapFromClasses(users, chats)
	for _, u := range updates {
		if upd, ok := u.(*tg.UpdateStory); ok {
			return types.ParseStory(upd.Story, pm), nil
		}
	}
	return nil, ErrStoryNotInUpdates
}
