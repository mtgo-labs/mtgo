package telegram

import (
	"context"
	"fmt"

	"github.com/mtgo-labs/mtgo/telegram/params"
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
func (c *StoryMethods) SendStory(ctx context.Context, chatID int64, media tg.InputMediaClass, opts ...*SendStoryOption) (*types.Story, error) {
	c.Log.Debugf("SendStory chat_id=%d", chatID)
	if media == nil {
		return nil, ErrMediaRequired
	}

	peer, err := resolvePeer(ctx, c.Client, chatID)
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

	req := &tg.StoriesSendStoryRequest{
		Pinned:       opt.Pinned,
		Noforwards:   opt.NoForwards,
		Peer:         peer,
		Media:        media,
		RandomID:     c.RandomID(),
		PrivacyRules: opt.PrivacyRules,
	}
	if opt.Period != nil {
		req.Period = *opt.Period // flags.3, derived by SetFlags (bit 8 is albums)
	}
	req.SetFlags()

	rpc := c.Raw()
	result, err := rpc.StoriesSendStory(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("send story: %w", err)
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
func (c *StoryMethods) EditStoryCaption(ctx context.Context, chatID int64, storyID int32, caption string) (*types.Story, error) {
	c.Log.Debugf("EditStoryCaption chat_id=%d story_id=%d", chatID, storyID)
	if caption == "" {
		return nil, ErrCaptionRequired
	}

	peer, err := resolvePeer(ctx, c.Client, chatID)
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
		return nil, fmt.Errorf("edit story: %w", err)
	}

	return extractStoryFromUpdates(result)
}

// EditStoryMedia replaces the media content of an existing story identified by
// storyID owned by chatID. The media parameter must be non-nil.
//
// Returns the updated Story on success, or an error if the media is nil,
// the peer cannot be resolved, or the Telegram API returns an error.
func (c *StoryMethods) EditStoryMedia(ctx context.Context, chatID int64, storyID int32, media tg.InputMediaClass) (*types.Story, error) {
	c.Log.Debugf("EditStoryMedia chat_id=%d story_id=%d", chatID, storyID)
	if media == nil {
		return nil, ErrMediaRequired
	}

	peer, err := resolvePeer(ctx, c.Client, chatID)
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
		return nil, fmt.Errorf("edit story: %w", err)
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
func (c *StoryMethods) DeleteStories(ctx context.Context, chatID int64, storyIDs []int32) error {
	c.Log.Debugf("DeleteStories chat_id=%d story_ids=%v", chatID, storyIDs)
	if len(storyIDs) == 0 {
		return ErrStoryIDsRequired
	}

	peer, err := resolvePeer(ctx, c.Client, chatID)
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
func (c *StoryMethods) GetStories(ctx context.Context, userID int64, storyIDs []int32) ([]*types.Story, error) {
	c.Log.Debugf("GetStories user_id=%d count=%d", userID, len(storyIDs))
	if len(storyIDs) == 0 {
		return nil, ErrStoryIDsRequired
	}

	user, err := resolveUserID(ctx, c.Client, userID)
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
		return nil, fmt.Errorf("get stories by id: %w", err)
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
func (c *StoryMethods) GetChatStories(ctx context.Context, chatID int64) ([]*types.Story, error) {
	c.Log.Debugf("GetChatStories chat_id=%d", chatID)
	peer, err := resolvePeer(ctx, c.Client, chatID)
	if err != nil {
		return nil, fmt.Errorf("resolve peer: %w", err)
	}

	req := &tg.StoriesGetPeerStoriesRequest{
		Peer: peer,
	}

	rpc := c.Raw()
	result, err := rpc.StoriesGetPeerStories(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("get peer stories: %w", err)
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
func (c *StoryMethods) GetStoryViews(ctx context.Context, chatID int64, storyIDs []int32) ([]*tg.StoryViews, error) {
	c.Log.Debugf("GetStoryViews chat_id=%d count=%d", chatID, len(storyIDs))
	if len(storyIDs) == 0 {
		return nil, ErrStoryIDsRequired
	}

	peer, err := resolvePeer(ctx, c.Client, chatID)
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
		return nil, fmt.Errorf("get stories views: %w", err)
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
func (c *StoryMethods) ForwardStory(ctx context.Context, targetChatID int64, sourceChatID int64, storyID int32) (*types.Message, error) {
	c.Log.Debugf("ForwardStory to=%d from=%d", targetChatID, sourceChatID)
	targetPeer, err := resolvePeer(ctx, c.Client, targetChatID)
	if err != nil {
		return nil, fmt.Errorf("resolve target peer: %w", err)
	}

	sourcePeer, err := resolvePeer(ctx, c.Client, sourceChatID)
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
		return nil, fmt.Errorf("send media: %w", err)
	}

	return extractSingleMessage(result, c)
}

// PinChatStories pins the specified stories to the top of the chat's story list.
// The storyIDs parameter specifies which stories to pin.
//
// Returns an error if the peer cannot be resolved or the Telegram API returns an error.
func (c *StoryMethods) PinChatStories(ctx context.Context, chatID int64, storyIDs []int32) error {
	peer, err := resolvePeer(ctx, c.Client, chatID)
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
func (c *StoryMethods) ReadChatStories(ctx context.Context, chatID int64, storyIDs []int32) error {
	peer, err := resolvePeer(ctx, c.Client, chatID)
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

// ReadStories marks the stories of a peer as read up to the given story ID.
func (c *StoryMethods) ReadStories(ctx context.Context, peerID int64, maxStoryID int32) error {
	peer, err := c.ResolvePeer(ctx, peerID)
	if err != nil {
		return err
	}
	_, err = c.Raw().StoriesReadStories(ctx, &tg.StoriesReadStoriesRequest{
		Peer:  peer,
		MaxID: maxStoryID,
	})
	return err
}

// EditStoryPrivacy replaces the privacy rules of a story and returns the
// updated story.
func (c *StoryMethods) EditStoryPrivacy(ctx context.Context, peerID int64, storyID int32, opts ...*params.EditPrivacy) (*types.Story, error) {
	peer, err := c.ResolvePeer(ctx, peerID)
	if err != nil {
		return nil, err
	}
	o := params.GetOptDef(&params.EditPrivacy{}, opts...)
	result, err := c.Raw().StoriesEditStory(ctx, &tg.StoriesEditStoryRequest{
		Peer:         peer,
		ID:           storyID,
		PrivacyRules: storyPrivacyRules(o),
	})
	if err != nil {
		return nil, fmt.Errorf("edit story: %w", err)
	}
	return extractStoryFromUpdates(result)
}

// SendStoryReaction reacts to a story with an emoji.
func (c *StoryMethods) SendStoryReaction(ctx context.Context, peerID int64, storyID int32, opts ...*params.React) error {
	o := params.GetOptDef(&params.React{}, opts...)
	peer, err := c.ResolvePeer(ctx, peerID)
	if err != nil {
		return err
	}
	_, err = c.Raw().StoriesSendReaction(ctx, &tg.StoriesSendReactionRequest{
		Peer:    peer,
		StoryID: storyID,
		Reaction: &tg.ReactionEmoji{
			Emoticon: o.Emoji,
		},
	})
	return err
}

// parseStoriesResult converts a stories result page into parsed stories.
func parseStoriesResult(result *tg.StoriesStories) []*types.Story {
	stories := make([]*types.Story, 0, len(result.Stories))
	pm := types.NewPeerMapFromClasses(result.Users, result.Chats)
	for _, s := range result.Stories {
		if parsed := types.ParseStory(s, pm); parsed != nil {
			stories = append(stories, parsed)
		}
	}
	return stories
}

// GetPinnedStories returns the stories a peer pinned to the top of its
// profile. offsetID paginates by story ID; a limit of 0 or less defaults to
// 10.
func (c *StoryMethods) GetPinnedStories(ctx context.Context, chatID int64, offsetID int32, limit int) ([]*types.Story, error) {
	c.Log.Debugf("GetPinnedStories chat_id=%d", chatID)
	peer, err := resolvePeer(ctx, c.Client, chatID)
	if err != nil {
		return nil, fmt.Errorf("resolve peer: %w", err)
	}
	result, err := c.Raw().StoriesGetPinnedStories(ctx, &tg.StoriesGetPinnedStoriesRequest{
		Peer:     peer,
		OffsetID: offsetID,
		Limit:    clampStoryLimit(limit),
	})
	if err != nil {
		return nil, fmt.Errorf("get pinned stories: %w", err)
	}
	return parseStoriesResult(result), nil
}

// GetArchivedStories returns the archived stories of a peer. offsetID
// paginates by story ID; a limit of 0 or less defaults to 10.
func (c *StoryMethods) GetArchivedStories(ctx context.Context, chatID int64, offsetID int32, limit int) ([]*types.Story, error) {
	c.Log.Debugf("GetArchivedStories chat_id=%d", chatID)
	peer, err := resolvePeer(ctx, c.Client, chatID)
	if err != nil {
		return nil, fmt.Errorf("resolve peer: %w", err)
	}
	result, err := c.Raw().StoriesGetStoriesArchive(ctx, &tg.StoriesGetStoriesArchiveRequest{
		Peer:     peer,
		OffsetID: offsetID,
		Limit:    clampStoryLimit(limit),
	})
	if err != nil {
		return nil, fmt.Errorf("get archived stories: %w", err)
	}
	return parseStoriesResult(result), nil
}

func clampStoryLimit(limit int) int32 {
	if limit <= 0 || limit > 100 {
		return 10
	}
	return int32(limit)
}

// StoriesFeedEntry is one peer's stories inside the all-stories feed.
type StoriesFeedEntry struct {
	// Chat is the peer that owns the stories.
	Chat *types.Chat
	// Stories are the peer's active stories.
	Stories []*types.Story
}

// StoriesFeed is one page of the all-stories feed of the logged-in user's
// contacts.
type StoriesFeed struct {
	// Count is the total number of peers with active stories.
	Count int32
	// State is the feed cursor; pass it back to receive further pages.
	State string
	// HasMore reports whether more pages follow.
	HasMore bool
	// StealthMode is the current stealth-mode state of the feed.
	StealthMode *tg.StoriesStealthMode
	// Peers holds each story-owning peer with its stories.
	Peers []StoriesFeedEntry
}

// GetAllStories returns the all-stories feed of the logged-in user's
// contacts. Pass the previous page's State (and next=true) to continue;
// hidden=true lists contacts hidden from the feed. Returns nil when the feed
// state is unchanged since the last call.
func (c *StoryMethods) GetAllStories(ctx context.Context, state string, next, hidden bool) (*StoriesFeed, error) {
	c.Log.Debugf("GetAllStories state=%q next=%v hidden=%v", state, next, hidden)
	result, err := c.Raw().StoriesGetAllStories(ctx, &tg.StoriesGetAllStoriesRequest{
		Next:   next,
		Hidden: hidden,
		State:  state,
	})
	if err != nil {
		return nil, fmt.Errorf("get all stories: %w", err)
	}
	switch v := result.(type) {
	case *tg.StoriesAllStoriesNotModified:
		return nil, nil
	case *tg.StoriesAllStories:
		pm := types.NewPeerMapFromClasses(v.Users, v.Chats)
		feed := &StoriesFeed{
			Count:       v.Count,
			State:       v.State,
			HasMore:     v.HasMore,
			StealthMode: v.StealthMode,
			Peers:       make([]StoriesFeedEntry, 0, len(v.PeerStories)),
		}
		for _, ps := range v.PeerStories {
			owned, ok := ps.(*tg.PeerStories)
			if !ok {
				continue
			}
			entry := StoriesFeedEntry{Chat: types.ParseChatFromPeer(owned.Peer, pm)}
			for _, s := range owned.Stories {
				if parsed := types.ParseStory(s, pm); parsed != nil {
					entry.Stories = append(entry.Stories, parsed)
				}
			}
			feed.Peers = append(feed.Peers, entry)
		}
		return feed, nil
	default:
		return nil, fmt.Errorf("get all stories: unexpected result %T", result)
	}
}

// ActivateStealthMode hides the logged-in user's story views from others:
// past hides views made during the last minutes, future hides upcoming views
// for a window. Stealth mode has a cooldown between activations.
func (c *StoryMethods) ActivateStealthMode(ctx context.Context, past, future bool) error {
	c.Log.Debugf("ActivateStealthMode past=%v future=%v", past, future)
	_, err := c.Raw().StoriesActivateStealthMode(ctx, &tg.StoriesActivateStealthModeRequest{
		Past:   past,
		Future: future,
	})
	if err != nil {
		return fmt.Errorf("activate stealth mode: %w", err)
	}
	return nil
}

// GetRemainingStoryCount returns how many stories the logged-in user can
// still post today on their own profile.
func (c *StoryMethods) GetRemainingStoryCount(ctx context.Context) (int32, error) {
	c.Log.Debug("GetRemainingStoryCount")
	result, err := c.Raw().StoriesCanSendStory(ctx, &tg.StoriesCanSendStoryRequest{
		Peer: &tg.InputPeerSelf{},
	})
	if err != nil {
		return 0, fmt.Errorf("get remaining story count: %w", err)
	}
	return result.CountRemains, nil
}

// GetChatsToSendStories returns the chats where the logged-in user may post
// stories (own profile plus admin channels).
func (c *StoryMethods) GetChatsToSendStories(ctx context.Context) ([]*types.Chat, error) {
	c.Log.Debug("GetChatsToSendStories")
	result, err := c.Raw().StoriesGetChatsToSend(ctx)
	if err != nil {
		return nil, fmt.Errorf("get chats to send stories: %w", err)
	}
	var raw []tg.ChatClass
	switch v := result.(type) {
	case *tg.MessagesChats:
		raw = v.Chats
	case *tg.MessagesChatsSlice:
		raw = v.Chats
	default:
		return nil, fmt.Errorf("get chats to send stories: unexpected result %T", result)
	}
	chats := make([]*types.Chat, 0, len(raw))
	for _, ch := range raw {
		if parsed := types.ParseChatFromChat(ch); parsed != nil {
			chats = append(chats, parsed)
		}
	}
	return chats, nil
}

// ExportStoryLink returns the public t.me link of a story.
func (c *StoryMethods) ExportStoryLink(ctx context.Context, chatID int64, storyID int32) (string, error) {
	c.Log.Debugf("ExportStoryLink chat_id=%d story_id=%d", chatID, storyID)
	peer, err := resolvePeer(ctx, c.Client, chatID)
	if err != nil {
		return "", fmt.Errorf("resolve peer: %w", err)
	}
	result, err := c.Raw().StoriesExportStoryLink(ctx, &tg.StoriesExportStoryLinkRequest{
		Peer: peer,
		ID:   storyID,
	})
	if err != nil {
		return "", fmt.Errorf("export story link: %w", err)
	}
	return result.Link, nil
}

// ToggleStoriesHidden hides (or unhides) a peer's stories from the logged-in
// user's main feed.
func (c *StoryMethods) ToggleStoriesHidden(ctx context.Context, chatID int64, hidden bool) error {
	c.Log.Debugf("ToggleStoriesHidden chat_id=%d hidden=%v", chatID, hidden)
	peer, err := resolvePeer(ctx, c.Client, chatID)
	if err != nil {
		return fmt.Errorf("resolve peer: %w", err)
	}
	_, err = c.Raw().StoriesTogglePeerStoriesHidden(ctx, &tg.StoriesTogglePeerStoriesHiddenRequest{
		Peer:   peer,
		Hidden: hidden,
	})
	if err != nil {
		return fmt.Errorf("toggle stories hidden: %w", err)
	}
	return nil
}

// ToggleAllStoriesHidden hides (or unhides) every contact's stories from the
// logged-in user's main feed.
func (c *StoryMethods) ToggleAllStoriesHidden(ctx context.Context, hidden bool) error {
	c.Log.Debugf("ToggleAllStoriesHidden hidden=%v", hidden)
	_, err := c.Raw().StoriesToggleAllStoriesHidden(ctx, &tg.StoriesToggleAllStoriesHiddenRequest{
		Hidden: hidden,
	})
	if err != nil {
		return fmt.Errorf("toggle all stories hidden: %w", err)
	}
	return nil
}

// StoryViewersPage is one page of a story's viewer list.
type StoryViewersPage struct {
	// Count is the total number of viewers.
	Count int32
	// ViewsCount, ForwardsCount, and ReactionsCount are the story's totals.
	ViewsCount     int32
	ForwardsCount  int32
	ReactionsCount int32
	// Views are the viewers on this page; entries may be reactions, forwards,
	// or plain views depending on the requested ordering.
	Views []tg.StoryViewClass
	// NextOffset fetches the next page; empty on the last page.
	NextOffset string
}

// GetStoryViewers lists the viewers of one story. Set justContacts to skip
// non-contacts, reactionsFirst or forwardsFirst to order reactions or
// forwards ahead of plain views, and q to filter by name.
func (c *StoryMethods) GetStoryViewers(ctx context.Context, chatID, storyID int64, q string, justContacts, reactionsFirst, forwardsFirst bool, offset string, limit int) (*StoryViewersPage, error) {
	c.Log.Debugf("GetStoryViewers chat_id=%d story_id=%d", chatID, storyID)
	peer, err := resolvePeer(ctx, c.Client, chatID)
	if err != nil {
		return nil, fmt.Errorf("resolve peer: %w", err)
	}
	result, err := c.Raw().StoriesGetStoryViewsList(ctx, &tg.StoriesGetStoryViewsListRequest{
		JustContacts:   justContacts,
		ReactionsFirst: reactionsFirst,
		ForwardsFirst:  forwardsFirst,
		Peer:           peer,
		Q:              q,
		ID:             int32(storyID),
		Offset:         offset,
		Limit:          clampStoryLimit(limit),
	})
	if err != nil {
		return nil, fmt.Errorf("get story viewers: %w", err)
	}
	return &StoryViewersPage{
		Count:          result.Count,
		ViewsCount:     result.ViewsCount,
		ForwardsCount:  result.ForwardsCount,
		ReactionsCount: result.ReactionsCount,
		Views:          result.Views,
		NextOffset:     result.NextOffset,
	}, nil
}

// StoryReactionsPage is one page of the reactions to a story.
type StoryReactionsPage struct {
	// Count is the total number of reactions.
	Count int32
	// Reactions are the reactions on this page.
	Reactions []tg.StoryReactionClass
	// NextOffset fetches the next page; empty on the last page.
	NextOffset string
}

// GetStoryReactions lists reactions to one story; pass a reaction to list
// only that reaction, and forwardsFirst to order forwards ahead.
func (c *StoryMethods) GetStoryReactions(ctx context.Context, chatID, storyID int64, reaction tg.ReactionClass, offset string, limit int, forwardsFirst bool) (*StoryReactionsPage, error) {
	c.Log.Debugf("GetStoryReactions chat_id=%d story_id=%d", chatID, storyID)
	peer, err := resolvePeer(ctx, c.Client, chatID)
	if err != nil {
		return nil, fmt.Errorf("resolve peer: %w", err)
	}
	result, err := c.Raw().StoriesGetStoryReactionsList(ctx, &tg.StoriesGetStoryReactionsListRequest{
		ForwardsFirst: forwardsFirst,
		Peer:          peer,
		ID:            int32(storyID),
		Reaction:      reaction,
		Offset:        offset,
		Limit:         clampStoryLimit(limit),
	})
	if err != nil {
		return nil, fmt.Errorf("get story reactions: %w", err)
	}
	return &StoryReactionsPage{
		Count:      result.Count,
		Reactions:  result.Reactions,
		NextOffset: result.NextOffset,
	}, nil
}

// ReportStory reports stories that violate the terms of service. option is a
// reason identifier obtained from the report options; message adds optional
// context.
func (c *StoryMethods) ReportStory(ctx context.Context, chatID int64, storyIDs []int32, option []byte, message string) error {
	c.Log.Debugf("ReportStory chat_id=%d count=%d", chatID, len(storyIDs))
	peer, err := resolvePeer(ctx, c.Client, chatID)
	if err != nil {
		return fmt.Errorf("resolve peer: %w", err)
	}
	_, err = c.Raw().StoriesReport(ctx, &tg.StoriesReportRequest{
		Peer:    peer,
		ID:      storyIDs,
		Option:  option,
		Message: message,
	})
	if err != nil {
		return fmt.Errorf("report story: %w", err)
	}
	return nil
}
