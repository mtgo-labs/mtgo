package types

import (
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
	ID         int32
	Date       time.Time
	ExpireDate time.Time
	Caption    string
	FromID     int64
	Out        bool
	Pinned     bool
	Public     bool
	Edited     bool
	Noforwards bool
	Media      Media
	binder     Binder
}

func (s *Story) SetBinder(b Binder) {
	s.binder = b
}

func (s *Story) Reply(text string, opts ...*params.SendMessage) (*Message, error) {
	if s.binder == nil {
		return nil, ErrNoBinder
	}
	return s.binder.BoundStoryReply(s.FromID, s.ID, text, opts...)
}

func (s *Story) ReplyText(text string, opts ...*params.SendMessage) (*Message, error) {
	return s.Reply(text, opts...)
}

func (s *Story) ReplyMedia(media tg.InputMediaClass, caption string, opts ...*params.SendMessage) (*Message, error) {
	if s.binder == nil {
		return nil, ErrNoBinder
	}
	return s.binder.BoundStoryReplyMedia(s.FromID, s.ID, media, caption, opts...)
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
	if s.binder == nil {
		return nil, ErrNoBinder
	}
	return nil, s.binder.BoundStub("Story.ReplyMediaGroup")
}

func (s *Story) Forward(toChatID int64) (*Message, error) {
	if s.binder == nil {
		return nil, ErrNoBinder
	}
	return s.binder.BoundStoryForward(s.FromID, s.ID, toChatID)
}

func (s *Story) Copy(toChatID int64) (*Message, error) {
	return s.Forward(toChatID)
}

func (s *Story) Delete() error {
	if s.binder == nil {
		return ErrNoBinder
	}
	return s.binder.BoundStoryDelete(s.FromID, s.ID)
}

func (s *Story) EditCaption(caption string) (*Story, error) {
	if s.binder == nil {
		return nil, ErrNoBinder
	}
	return s.binder.BoundStoryEditCaption(s.FromID, s.ID, caption)
}

func (s *Story) EditMedia(media tg.InputMediaClass) (*Story, error) {
	if s.binder == nil {
		return nil, ErrNoBinder
	}
	return s.binder.BoundStoryEditMedia(s.FromID, s.ID, media)
}

func (s *Story) EditPrivacy() (*Story, error) {
	if s.binder == nil {
		return nil, ErrNoBinder
	}
	return s.binder.BoundStoryEditPrivacy(s.FromID, s.ID)
}

func (s *Story) React(emoji string) error {
	if s.binder == nil {
		return ErrNoBinder
	}
	return s.binder.BoundStoryReact(s.FromID, s.ID, emoji)
}

func (s *Story) Download(opts ...*params.Download) ([]byte, error) {
	if s.binder == nil {
		return nil, ErrNoBinder
	}
	return s.binder.BoundStoryDownload(s.FromID, s.ID, opts...)
}

func (s *Story) Read() error {
	if s.binder == nil {
		return ErrNoBinder
	}
	return s.binder.BoundStoryRead(s.FromID, s.ID)
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
			ID:         v.ID,
			Date:       time.Unix(int64(v.Date), 0),
			Out:        v.Out,
			Pinned:     v.Pinned,
			Public:     v.Public,
			Edited:     v.Edited,
			Noforwards: v.Noforwards,
			Media:      ParseMedia(v.Media),
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
