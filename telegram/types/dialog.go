package types

import (
	"time"

	"github.com/mtgo-labs/mtgo/tg"
)

// Dialog represents a single conversation entry in the user's chat list,
// including unread counts, pin state, and the associated chat.
//
// Example:
//
//	dialogs, _ := client.GetDialogs(ctx, 0, 20)
//	for _, d := range dialogs {
//	    fmt.Printf("Chat %d: %d unread, pinned=%v\n", d.Chat.ID, d.UnreadMessagesCount, d.IsPinned)
//	}
type Dialog struct {
	Chat                    *Chat
	TopMessage              *Message
	LastReadInboxMessageID  int32
	LastReadOutboxMessageID int32
	UnreadMessagesCount     int32
	UnreadMentionsCount     int32
	UnreadReactionsCount    int32
	UnreadPollVoteCount     int32
	UnreadMark              bool
	IsPinned                bool
	FolderID                int32
	TTLPeriod               int32
}

// PeerInfo holds the identity and type of a Telegram peer (user, group, or channel).
type PeerInfo struct {
	// ID is the numeric identifier of the peer. Negative for groups and channels.
	ID int64
	// Type indicates whether the peer is a private chat, group, or channel.
	Type ChatType
	// AccessHash is the access hash required to interact with the peer, when available.
	AccessHash int64
}

// FolderBinder provides folder management operations bound to a client.
type FolderBinder interface {
	BoundGetFolders() ([]tg.DialogFilterClass, error)
	BoundEditFolder(filter *tg.DialogFilter) error
	BoundDeleteFolder(folderID int32) error
	BoundIncludeChat(folderID int32, chatID int64) error
	BoundExcludeChat(folderID int32, chatID int64) error
	BoundUpdateFolderColor(folderID int32, color int32) error
	BoundPinChatInFolder(folderID int32, chatID int64) error
	BoundRemoveChatFromFolder(folderID int32, chatID int64) error
}

// Folder represents a chat folder (dialog filter) with inclusion/exclusion rules,
// pinned chats, and management methods via a FolderBinder.
//
// Example:
//
//	folders, _ := client.GetFolders(ctx)
//	for _, f := range folders {
//	    fmt.Printf("Folder: %s (icon: %s, included: %d chats)\n", f.Name, f.Icon, len(f.IncludedChats))
//	}
type Folder struct {
	ID                 int32
	Name               string
	Entities           []*MessageEntity
	AnimateCustomEmoji bool
	Icon               string
	Color              int32
	IsShareable        bool
	PinnedChats        []*Chat
	IncludedChats      []*Chat
	ExcludedChats      []*Chat
	ExcludeMuted       bool
	ExcludeRead        bool
	ExcludeArchived    bool
	IncludeContacts    bool
	IncludeNonContacts bool
	IncludeBots        bool
	IncludeGroups      bool
	IncludeChannels    bool
	binder             FolderBinder
}

func (f *Folder) SetBinder(b FolderBinder) {
	f.binder = b
}

func (f *Folder) Delete() error {
	if f.binder == nil {
		return ErrNoBinder
	}
	return f.binder.BoundDeleteFolder(f.ID)
}

func (f *Folder) Edit(name string, icon string) error {
	if f.binder == nil {
		return ErrNoBinder
	}
	raw, err := f.toDialogFilter()
	if err != nil {
		return err
	}
	if name != "" {
		f.Name = name
	}
	if icon != "" {
		f.Icon = icon
	}
	raw.Title.Text = f.Name
	if f.Icon != "" {
		raw.Emoticon = f.Icon
	}
	return f.binder.BoundEditFolder(raw)
}

func (f *Folder) IncludeChat(chatID int64) error {
	if f.binder == nil {
		return ErrNoBinder
	}
	return f.binder.BoundIncludeChat(f.ID, chatID)
}

func (f *Folder) ExcludeChat(chatID int64) error {
	if f.binder == nil {
		return ErrNoBinder
	}
	return f.binder.BoundExcludeChat(f.ID, chatID)
}

func (f *Folder) UpdateColor(color int32) error {
	if f.binder == nil {
		return ErrNoBinder
	}
	return f.binder.BoundUpdateFolderColor(f.ID, color)
}

func (f *Folder) PinChat(chatID int64) error {
	if f.binder == nil {
		return ErrNoBinder
	}
	return f.binder.BoundPinChatInFolder(f.ID, chatID)
}

func (f *Folder) RemoveChat(chatID int64) error {
	if f.binder == nil {
		return ErrNoBinder
	}
	return f.binder.BoundRemoveChatFromFolder(f.ID, chatID)
}

func (f *Folder) toDialogFilter() (*tg.DialogFilter, error) {
	df := &tg.DialogFilter{
		ID:              f.ID,
		Contacts:        f.IncludeContacts,
		NonContacts:     f.IncludeNonContacts,
		Groups:          f.IncludeGroups,
		Broadcasts:      f.IncludeChannels,
		Bots:            f.IncludeBots,
		ExcludeMuted:    f.ExcludeMuted,
		ExcludeRead:     f.ExcludeRead,
		ExcludeArchived: f.ExcludeArchived,
	}
	df.Title.Text = f.Name
	if f.Icon != "" {
		df.Emoticon = f.Icon
	}
	if f.Color != 0 {
		df.Color = f.Color
	}
	return df, nil
}

// EmojiStatus represents a custom emoji status displayed next to a user's name,
// such as a Premium or collectible emoji. May be time-limited.
type EmojiStatus struct {
	CustomEmojiID        int64
	UntilDate            time.Time
	Title                string
	GiftID               int64
	PatternCustomEmojiID int64
	CenterColor          int32
	EdgeColor            int32
	PatternColor         int32
	TextColor            int32
}

// ChatColor represents the accent color and background emoji chosen for a chat.
type ChatColor struct {
	// Color is the palette index for the chat's accent color.
	Color int32
	// BackgroundEmojiID is the custom emoji document ID used as the chat background pattern.
	BackgroundEmojiID int64
}

// Restriction represents a platform-specific restriction applied to a chat or user,
// such as content warnings or regional blocks.
type Restriction struct {
	// Platform is the platform the restriction applies to (e.g. "ios", "android").
	Platform string
	// Reason is the machine-readable reason code for the restriction (e.g. "terms").
	Reason string
	// Text is the human-readable explanation of the restriction.
	Text string
}

// ParseDialog converts an MTProto DialogTL into a Dialog.
// Returns nil if raw is nil.
func ParseDialog(raw *tg.Dialog, users map[int64]tg.UserClass, chats *PeerMap) *Dialog {
	if raw == nil {
		return nil
	}
	d := &Dialog{
		LastReadInboxMessageID:  raw.ReadInboxMaxID,
		LastReadOutboxMessageID: raw.ReadOutboxMaxID,
		UnreadMessagesCount:     raw.UnreadCount,
		UnreadMentionsCount:     raw.UnreadMentionsCount,
		UnreadReactionsCount:    raw.UnreadReactionsCount,
		UnreadPollVoteCount:     raw.UnreadPollVotesCount,
		UnreadMark:              raw.UnreadMark,
		IsPinned:                raw.Pinned,
	}
	if raw.FolderID != 0 {
		d.FolderID = raw.FolderID
	}
	if raw.TTLPeriod != 0 {
		d.TTLPeriod = raw.TTLPeriod
	}
	d.Chat = ParseChatFromPeer(raw.Peer, chats)
	return d
}

// ParseFolder converts an MTProto DialogFilterTL into a Folder.
// Returns nil if raw is nil.
func ParseFolder(raw *tg.DialogFilter) *Folder {
	if raw == nil {
		return nil
	}
	f := &Folder{
		ID:                 raw.ID,
		IncludeContacts:    raw.Contacts,
		IncludeNonContacts: raw.NonContacts,
		IncludeGroups:      raw.Groups,
		IncludeChannels:    raw.Broadcasts,
		IncludeBots:        raw.Bots,
		ExcludeMuted:       raw.ExcludeMuted,
		ExcludeRead:        raw.ExcludeRead,
		ExcludeArchived:    raw.ExcludeArchived,
	}
	if raw.Title != nil {
		f.Name = raw.Title.Text
		for _, ent := range raw.Title.Entities {
			if e := ParseMessageEntity(ent); e != nil {
				f.Entities = append(f.Entities, e)
			}
		}
	}
	if !raw.TitleNoanimate {
		f.AnimateCustomEmoji = true
	}
	if raw.Emoticon != "" {
		f.Icon = raw.Emoticon
	}
	if raw.Color != 0 {
		f.Color = raw.Color
	}
	for _, p := range raw.PinnedPeers {
		if c := chatFromInputPeer(p); c != nil {
			f.PinnedChats = append(f.PinnedChats, c)
		}
	}
	for _, p := range raw.IncludePeers {
		if c := chatFromInputPeer(p); c != nil {
			f.IncludedChats = append(f.IncludedChats, c)
		}
	}
	for _, p := range raw.ExcludePeers {
		if c := chatFromInputPeer(p); c != nil {
			f.ExcludedChats = append(f.ExcludedChats, c)
		}
	}
	return f
}

// ParseEmojiStatus converts an MTProto EmojiStatusClass into an EmojiStatus.
// Returns nil if raw is nil.
func ParseEmojiStatus(raw tg.EmojiStatusClass) *EmojiStatus {
	if raw == nil {
		return nil
	}
	switch s := raw.(type) {
	case *tg.EmojiStatus:
		e := &EmojiStatus{CustomEmojiID: s.DocumentID}
		if s.Until != 0 {
			e.UntilDate = time.Unix(int64(s.Until), 0)
		}
		return e
	case *tg.EmojiStatusCollectible:
		e := &EmojiStatus{
			GiftID:               s.CollectibleID,
			CustomEmojiID:        s.DocumentID,
			Title:                s.Title,
			PatternCustomEmojiID: s.PatternDocumentID,
			CenterColor:          s.CenterColor,
			EdgeColor:            s.EdgeColor,
			PatternColor:         s.PatternColor,
			TextColor:            s.TextColor,
		}
		if s.Until != 0 {
			e.UntilDate = time.Unix(int64(s.Until), 0)
		}
		return e
	}
	return nil
}

// ParseChatColor converts an MTProto PeerColorTL into a ChatColor.
// Returns nil if raw is nil.
func ParseChatColor(raw *tg.PeerColor) *ChatColor {
	if raw == nil {
		return nil
	}
	c := &ChatColor{}
	if raw.Color != 0 {
		c.Color = raw.Color
	}
	if raw.BackgroundEmojiID != 0 {
		c.BackgroundEmojiID = raw.BackgroundEmojiID
	}
	return c
}

// ParseChatColorFromPeer converts an MTProto PeerColorClass into a ChatColor.
// Returns nil if raw is nil.
func ParseChatColorFromPeer(raw tg.PeerColorClass) *ChatColor {
	if raw == nil {
		return nil
	}
	if pc, ok := raw.(*tg.PeerColor); ok {
		return ParseChatColor(pc)
	}
	return nil
}

// ParseRestriction converts an MTProto RestrictionReason into a Restriction.
// Returns nil if raw is nil.
func ParseRestriction(raw *tg.RestrictionReason) *Restriction {
	if raw == nil {
		return nil
	}
	return &Restriction{
		Platform: raw.Platform,
		Reason:   raw.Reason,
		Text:     raw.Text,
	}
}

func parseRestrictions(raw []*tg.RestrictionReason) []*Restriction {
	if raw == nil {
		return nil
	}
	out := make([]*Restriction, 0, len(raw))
	for _, r := range raw {
		if v := ParseRestriction(r); v != nil {
			out = append(out, v)
		}
	}
	return out
}

func chatFromInputPeer(peer tg.InputPeerClass) *Chat {
	if peer == nil {
		return nil
	}
	switch p := peer.(type) {
	case *tg.InputPeerUser:
		return &Chat{ID: p.UserID, Type: ChatTypePrivate, AccessHash: p.AccessHash}
	case *tg.InputPeerChat:
		return &Chat{ID: -p.ChatID, Type: ChatTypeGroup}
	case *tg.InputPeerChannel:
		return &Chat{ID: -p.ChannelID, Type: ChatTypeChannel, AccessHash: p.AccessHash}
	}
	return nil
}
