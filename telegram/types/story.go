package types

import (
	"context"
	"time"

	"github.com/mtgo-labs/mtgo/telegram/params"
	"github.com/mtgo-labs/mtgo/tg"
)

// Story represents a Telegram story with its metadata and attached media.
// Stories are ephemeral posts visible for 24 hours on a user's or channel's profile.
//
// Example:
//
//	story := types.ParseStory(rawStoryItem, peerMap)
//	if story != nil {
//	    fmt.Printf("Story %d by user %d: %s\n", story.ID, story.FromID, story.Caption)
//	}
type Story struct {
	ID                  int32
	FromUser            *User
	SenderChat          *Chat
	Date                time.Time
	Chat                *Chat
	ForwardFrom         *User
	ForwardSenderName   string
	ForwardFromChat     *Chat
	ForwardFromStoryID  int32
	ExpireDate          time.Time
	Media               Media
	HasProtectedContent bool
	Photo               *Photo
	Video               *DocumentMedia
	Edited              bool
	Pinned              bool
	Public              bool
	CloseFriends        bool
	Contacts            bool
	SelectedContacts    bool
	Caption             string
	CaptionEntities     []*MessageEntity
	Views               int32
	Forwards            int32
	Outgoing            bool
	Reactions           []Reaction
	ReactionsCount      int32
	Skipped             bool
	Deleted             bool
	MediaAreas          []*MediaArea
	Privacy             StoriesPrivacyRules
	AllowedUsers        []*User
	DisallowedUsers     []*User
	Raw                 tg.StoryItemClass
	FromID              int64
	binder              StoryBinder
}

func (s *Story) SetBinder(b StoryBinder) {
	s.binder = b
}

func (s *Story) Reply(text string, opts ...*params.SendMessage) (*Message, error) {
	return s.ReplyContext(context.Background(), text, opts...)
}

func (s *Story) ReplyContext(ctx context.Context, text string, opts ...*params.SendMessage) (*Message, error) {
	if s.binder == nil {
		return nil, ErrNoBinder
	}
	return s.binder.BoundStoryReply(ctx, s.FromID, s.ID, text, opts...)
}

func (s *Story) ReplyText(text string, opts ...*params.SendMessage) (*Message, error) {
	return s.Reply(text, opts...)
}

func (s *Story) ReplyMedia(media tg.InputMediaClass, caption string, opts ...*params.SendMessage) (*Message, error) {
	return s.ReplyMediaContext(context.Background(), media, caption, opts...)
}

func (s *Story) ReplyMediaContext(ctx context.Context, media tg.InputMediaClass, caption string, opts ...*params.SendMessage) (*Message, error) {
	if s.binder == nil {
		return nil, ErrNoBinder
	}
	return s.binder.BoundStoryReplyMedia(ctx, s.FromID, s.ID, media, caption, opts...)
}

func (s *Story) ReplyPhoto(fileID, accessHash int64, caption string, opts ...*params.SendMessage) (*Message, error) {
	return s.ReplyMedia(&tg.InputMediaPhoto{ID: &tg.InputPhoto{ID: fileID, AccessHash: accessHash}}, caption, opts...)
}

func (s *Story) ReplyAnimation(fileID, accessHash int64, caption string, opts ...*params.SendMessage) (*Message, error) {
	return s.ReplyMedia(&tg.InputMediaDocument{ID: &tg.InputDocument{ID: fileID, AccessHash: accessHash}}, caption, opts...)
}

func (s *Story) ReplyAudio(fileID, accessHash int64, caption string, opts ...*params.SendMessage) (*Message, error) {
	return s.ReplyMedia(&tg.InputMediaDocument{ID: &tg.InputDocument{ID: fileID, AccessHash: accessHash}}, caption, opts...)
}

func (s *Story) ReplyVideo(fileID, accessHash int64, caption string, opts ...*params.SendMessage) (*Message, error) {
	return s.ReplyMedia(&tg.InputMediaDocument{ID: &tg.InputDocument{ID: fileID, AccessHash: accessHash}}, caption, opts...)
}

func (s *Story) ReplyVideoNote(fileID, accessHash int64, opts ...*params.SendMessage) (*Message, error) {
	return s.ReplyMedia(&tg.InputMediaDocument{ID: &tg.InputDocument{ID: fileID, AccessHash: accessHash}}, "", opts...)
}

func (s *Story) ReplyVoice(fileID, accessHash int64, caption string, opts ...*params.SendMessage) (*Message, error) {
	return s.ReplyMedia(&tg.InputMediaDocument{ID: &tg.InputDocument{ID: fileID, AccessHash: accessHash}}, caption, opts...)
}

func (s *Story) ReplySticker(fileID, accessHash int64, opts ...*params.SendMessage) (*Message, error) {
	return s.ReplyMedia(&tg.InputMediaDocument{ID: &tg.InputDocument{ID: fileID, AccessHash: accessHash}}, "", opts...)
}

func (s *Story) ReplyCachedMedia(fileID, accessHash int64, caption string, opts ...*params.SendMessage) (*Message, error) {
	return s.ReplyMedia(&tg.InputMediaDocument{ID: &tg.InputDocument{ID: fileID, AccessHash: accessHash}}, caption, opts...)
}

func (s *Story) ReplyMediaGroup(media []tg.InputMediaClass, opts ...*params.SendMessage) ([]*Message, error) {
	return s.ReplyMediaGroupContext(context.Background(), media, opts...)
}

func (s *Story) ReplyMediaGroupContext(ctx context.Context, media []tg.InputMediaClass, opts ...*params.SendMessage) ([]*Message, error) {
	if s.binder == nil {
		return nil, ErrNoBinder
	}
	return nil, s.binder.BoundStub(ctx, "Story.ReplyMediaGroup")
}

func (s *Story) Forward(chatID int64, opts ...*params.StoryForward) (*Message, error) {
	return s.ForwardContext(context.Background(), chatID, opts...)
}

func (s *Story) ForwardContext(ctx context.Context, chatID int64, opts ...*params.StoryForward) (*Message, error) {
	if s.binder == nil {
		return nil, ErrNoBinder
	}
	return s.binder.BoundStoryForward(ctx, s.FromID, s.ID, chatID, opts...)
}

func (s *Story) Copy(chatID int64, opts ...*params.StoryForward) (*Message, error) {
	return s.Forward(chatID, opts...)
}

func (s *Story) Delete() error {
	return s.DeleteContext(context.Background())
}

func (s *Story) DeleteContext(ctx context.Context) error {
	if s.binder == nil {
		return ErrNoBinder
	}
	return s.binder.BoundStoryDelete(ctx, s.FromID, s.ID)
}

func (s *Story) EditCaption(caption string) (*Story, error) {
	return s.EditCaptionContext(context.Background(), caption)
}

func (s *Story) EditCaptionContext(ctx context.Context, caption string) (*Story, error) {
	if s.binder == nil {
		return nil, ErrNoBinder
	}
	return s.binder.BoundStoryEditCaption(ctx, s.FromID, s.ID, &params.EditCaption{Caption: caption})
}

func (s *Story) EditMedia(media tg.InputMediaClass) (*Story, error) {
	return s.EditMediaContext(context.Background(), media)
}

func (s *Story) EditMediaContext(ctx context.Context, media tg.InputMediaClass) (*Story, error) {
	if s.binder == nil {
		return nil, ErrNoBinder
	}
	return s.binder.BoundStoryEditMedia(ctx, s.FromID, s.ID, media)
}

func (s *Story) EditPrivacy(opts ...*params.EditPrivacy) (*Story, error) {
	return s.EditPrivacyContext(context.Background(), opts...)
}

func (s *Story) EditPrivacyContext(ctx context.Context, opts ...*params.EditPrivacy) (*Story, error) {
	if s.binder == nil {
		return nil, ErrNoBinder
	}
	return s.binder.BoundStoryEditPrivacy(ctx, s.FromID, s.ID, opts...)
}

func (s *Story) React(emoji string) error {
	return s.ReactContext(context.Background(), emoji)
}

func (s *Story) ReactContext(ctx context.Context, emoji string) error {
	if s.binder == nil {
		return ErrNoBinder
	}
	return s.binder.BoundStoryReact(ctx, s.FromID, s.ID, &params.React{Emoji: emoji})
}

func (s *Story) Download(opts ...*params.Download) ([]byte, error) {
	return s.DownloadContext(context.Background(), opts...)
}

func (s *Story) DownloadContext(ctx context.Context, opts ...*params.Download) ([]byte, error) {
	if s.binder == nil {
		return nil, ErrNoBinder
	}
	return s.binder.BoundStoryDownload(ctx, s.FromID, s.ID, opts...)
}

func (s *Story) Read() error {
	return s.ReadContext(context.Background())
}

func (s *Story) ReadContext(ctx context.Context) error {
	if s.binder == nil {
		return ErrNoBinder
	}
	return s.binder.BoundStoryRead(ctx, s.FromID, s.ID)
}

func (s *Story) View() error {
	return s.Read()
}

// ParseStory converts a TL story item into a Story. Returns nil for deleted
// or skipped stories, or when raw is nil.
func ParseStory(raw tg.StoryItemClass, _ *PeerMap) *Story {
	if raw == nil {
		return nil
	}
	switch v := raw.(type) {
	case *tg.StoryItemDeleted:
		return nil
	case *tg.StoryItemSkipped:
		return nil
	case *tg.StoryItem:
		s := &Story{
			ID:                  v.ID,
			Date:                time.Unix(int64(v.Date), 0),
			Outgoing:            v.Out,
			Pinned:              v.Pinned,
			Public:              v.Public,
			Edited:              v.Edited,
			HasProtectedContent: v.Noforwards,
			Media:               ParseMedia(v.Media),
			Raw:                 raw,
		}
		if v.ExpireDate != 0 {
			s.ExpireDate = time.Unix(int64(v.ExpireDate), 0)
		}
		if v.Caption != "" {
			s.Caption = v.Caption
		}
		if v.FromID != nil {
			if p, ok := v.FromID.(*tg.PeerUser); ok {
				s.FromID = p.UserID
			}
		}
		return s
	}
	return nil
}

// StoriesStealthMode represents the temporary stealth mode state for stories,
// indicating when stealth is active and when the cooldown expires.
//
// Example:
//
//	mode := types.ParseStoriesStealthMode(rawMode)
//	fmt.Printf("Stealth active until: %s\n", mode.ActiveUntilDate)
type StoriesStealthMode struct {
	ActiveUntilDate   time.Time
	CooldownUntilDate time.Time
}

// ParseStoriesStealthMode converts a TL StoriesStealthMode into a StoriesStealthMode.
// Returns nil if raw is nil.
//
// Example:
//
//	mode := types.ParseStoriesStealthMode(rawMode)
//	if mode != nil && !mode.ActiveUntilDate.IsZero() {
//	    fmt.Println("Stealth mode is currently active")
//	}
func ParseStoriesStealthMode(raw *tg.StoriesStealthMode) *StoriesStealthMode {
	if raw == nil {
		return nil
	}
	m := &StoriesStealthMode{}
	if raw.ActiveUntilDate != 0 {
		m.ActiveUntilDate = time.Unix(int64(raw.ActiveUntilDate), 0)
	}
	if raw.CooldownUntilDate != 0 {
		m.CooldownUntilDate = time.Unix(int64(raw.CooldownUntilDate), 0)
	}
	return m
}
