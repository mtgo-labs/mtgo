package types

import (
	"github.com/mtgo-labs/mtgo/tg"
)

// Dialog represents a single conversation entry in the user's chat list,
// including unread counts, pin state, and the associated peer.
//
// Example:
//
//	dialogs, _ := client.GetDialogs(ctx, 0, 20)
//	for _, d := range dialogs {
//	    fmt.Printf("Peer %d: %d unread, pinned=%v\n", d.Peer.ID, d.UnreadCount, d.Pinned)
//	}
type Dialog struct {
	// Peer identifies the chat, user, or channel this dialog represents.
	Peer *PeerInfo
	// TopMessage is the ID of the most recent message in this dialog.
	TopMessage int32
	// ReadInboxMaxID is the message ID up to which all incoming messages have been read.
	ReadInboxMaxID int32
	// ReadOutboxMaxID is the message ID up to which all outgoing messages have been read by the peer.
	ReadOutboxMaxID int32
	// UnreadCount is the number of unread incoming messages in this dialog.
	UnreadCount int32
	// UnreadMentionsCount is the number of unread messages that mention the current user.
	UnreadMentionsCount int32
	// UnreadReactionsCount is the number of unread reactions on the current user's messages.
	UnreadReactionsCount int32
	// Pinned indicates whether this dialog is pinned to the top of the chat list.
	Pinned bool
	// UnreadMark indicates the user has manually marked this dialog as unread.
	UnreadMark bool
	// FolderID is the chat folder this dialog belongs to, or zero for the main list.
	FolderID int32
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

type Folder struct {
	ID              int32
	Title           string
	Emoticon        string
	Color           int32
	Contacts        bool
	NonContacts     bool
	Groups          bool
	Broadcasts      bool
	Bots            bool
	ExcludeMuted    bool
	ExcludeRead     bool
	ExcludeArchived bool
	PinnedPeers     []*PeerInfo
	IncludePeers    []*PeerInfo
	ExcludePeers    []*PeerInfo
	binder          FolderBinder
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

func (f *Folder) Edit(title string, emoticon string) error {
	if f.binder == nil {
		return ErrNoBinder
	}
	raw, err := f.toDialogFilter()
	if err != nil {
		return err
	}
	if title != "" {
		f.Title = title
	}
	if emoticon != "" {
		f.Emoticon = emoticon
	}
	raw.Title.Text = f.Title
	if f.Emoticon != "" {
		raw.Emoticon = f.Emoticon
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
		Contacts:        f.Contacts,
		NonContacts:     f.NonContacts,
		Groups:          f.Groups,
		Broadcasts:      f.Broadcasts,
		Bots:            f.Bots,
		ExcludeMuted:    f.ExcludeMuted,
		ExcludeRead:     f.ExcludeRead,
		ExcludeArchived: f.ExcludeArchived,
	}
	df.Title.Text = f.Title
	if f.Emoticon != "" {
		df.Emoticon = f.Emoticon
	}
	if f.Color != 0 {
		df.Color = f.Color
	}
	return df, nil
}

// EmojiStatus represents a custom emoji status displayed next to a user's name,
// such as a Premium or collectible emoji. May be time-limited.
type EmojiStatus struct {
	// DocumentID is the custom emoji sticker document ID for the status.
	DocumentID int64
	// Until is the Unix timestamp when the status expires, or zero for permanent.
	Until int32
	// CollectibleID is the unique ID of the collectible emoji, if this is a collectible status.
	CollectibleID int64
	// Title is the display title of the collectible emoji status.
	Title string
	// Slug is the URL slug of the collectible emoji status.
	Slug string
	// IsCollectible indicates whether this emoji status is a collectible item.
	IsCollectible bool
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
func ParseDialog(raw *tg.Dialog) *Dialog {
	if raw == nil {
		return nil
	}
	d := &Dialog{
		TopMessage:           raw.TopMessage,
		ReadInboxMaxID:       raw.ReadInboxMaxID,
		ReadOutboxMaxID:      raw.ReadOutboxMaxID,
		UnreadCount:          raw.UnreadCount,
		UnreadMentionsCount:  raw.UnreadMentionsCount,
		UnreadReactionsCount: raw.UnreadReactionsCount,
		Pinned:               raw.Pinned,
		UnreadMark:           raw.UnreadMark,
	}
	d.Peer = parsePeerInfo(raw.Peer)
	if raw.FolderID != 0 {
		d.FolderID = raw.FolderID
	}
	return d
}

// ParseFolder converts an MTProto DialogFilterTL into a Folder.
// Returns nil if raw is nil.
func ParseFolder(raw *tg.DialogFilter) *Folder {
	if raw == nil {
		return nil
	}
	f := &Folder{
		ID:              raw.ID,
		Contacts:        raw.Contacts,
		NonContacts:     raw.NonContacts,
		Groups:          raw.Groups,
		Broadcasts:      raw.Broadcasts,
		Bots:            raw.Bots,
		ExcludeMuted:    raw.ExcludeMuted,
		ExcludeRead:     raw.ExcludeRead,
		ExcludeArchived: raw.ExcludeArchived,
	}
	if raw.Title != nil {
		f.Title = raw.Title.Text
	}
	if raw.Emoticon != "" {
		f.Emoticon = raw.Emoticon
	}
	if raw.Color != 0 {
		f.Color = raw.Color
	}
	for _, p := range raw.PinnedPeers {
		f.PinnedPeers = append(f.PinnedPeers, parseInputPeerInfo(p))
	}
	for _, p := range raw.IncludePeers {
		f.IncludePeers = append(f.IncludePeers, parseInputPeerInfo(p))
	}
	for _, p := range raw.ExcludePeers {
		f.ExcludePeers = append(f.ExcludePeers, parseInputPeerInfo(p))
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
		e := &EmojiStatus{DocumentID: s.DocumentID}
		if s.Until != 0 {
			e.Until = s.Until
		}
		return e
	case *tg.EmojiStatusCollectible:
		e := &EmojiStatus{
			CollectibleID: s.CollectibleID,
			DocumentID:    s.DocumentID,
			Title:         s.Title,
			Slug:          s.Slug,
			IsCollectible: true,
		}
		if s.Until != 0 {
			e.Until = s.Until
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

func parsePeerInfo(peer tg.PeerClass) *PeerInfo {
	if peer == nil {
		return nil
	}
	switch p := peer.(type) {
	case *tg.PeerUser:
		return &PeerInfo{ID: p.UserID, Type: ChatTypePrivate}
	case *tg.PeerChat:
		return &PeerInfo{ID: -p.ChatID, Type: ChatTypeGroup}
	case *tg.PeerChannel:
		return &PeerInfo{ID: -p.ChannelID, Type: ChatTypeChannel}
	}
	return nil
}

func parseInputPeerInfo(peer tg.InputPeerClass) *PeerInfo {
	if peer == nil {
		return nil
	}
	switch p := peer.(type) {
	case *tg.InputPeerUser:
		return &PeerInfo{ID: p.UserID, Type: ChatTypePrivate, AccessHash: p.AccessHash}
	case *tg.InputPeerChat:
		return &PeerInfo{ID: -p.ChatID, Type: ChatTypeGroup}
	case *tg.InputPeerChannel:
		return &PeerInfo{ID: -p.ChannelID, Type: ChatTypeChannel, AccessHash: p.AccessHash}
	}
	return nil
}
