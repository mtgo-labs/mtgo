// Package params defines option and parameter structs used to configure
// Telegram Bot API calls such as sending, editing, forwarding, copying,
// deleting, and pinning messages, as well as file upload/download progress.
package params

import (
	"fmt"

	tl "github.com/mtgo-labs/mtgo/tg"
)

// ParseMode controls how message text is interpreted by the Telegram API.
// Use it to select between plain text, Markdown, HTML, or to disable
// parsing entirely when sending or editing messages.
//
// Example:
//
//	mode := params.ParseModeHTML
//	msg, err := client.SendMessage(ctx, chatID, "<b>Bold text</b>", &params.SendMessage{ParseMode: mode})
type ParseMode string

const (
	// ParseModeDefault uses the client's default parsing behaviour (no
	// explicit mode is sent to the API).
	ParseModeDefault ParseMode = "default"

	// ParseModeMarkdown interprets message text as Markdown-formatted
	// content (legacy mode).
	ParseModeMarkdown ParseMode = "markdown"

	// ParseModeHTML interprets message text as HTML-formatted content.
	ParseModeHTML ParseMode = "html"

	// ParseModeDisabled disables all parsing; the message text is sent
	// verbatim without any formatting applied.
	ParseModeDisabled ParseMode = "disabled"

	// Markdown is a shorthand for ParseModeMarkdown.
	Markdown ParseMode = "markdown"
	// HTML is a shorthand for ParseModeHTML.
	HTML ParseMode = "html"
	// MarkdownV2 interprets message text as MarkdownV2-formatted content.
	MarkdownV2 ParseMode = "MarkdownV2"
	// Disabled is a shorthand for ParseModeDisabled.
	Disabled ParseMode = "disabled"
)

// String returns the raw string representation of the ParseMode value.
func (p ParseMode) String() string { return string(p) }

// GetOptDef returns the first valid option from opts, or def when opts is
// empty or the provided option is a zero-value. It panics when more than one
// option is passed, because each parameter should accept at most one explicit
// override. This generic helper is the backbone of the functional-options
// pattern used throughout the params package.
func GetOptDef[T comparable](def T, opts ...T) T {
	if len(opts) == 0 {
		return def
	}
	if len(opts) > 1 {
		panic(fmt.Sprintf("too many options: expected 0 or 1, got %d", len(opts)))
	}
	first := opts[0]
	if !validOpt(first) {
		return def
	}
	return first
}

func validOpt[T comparable](opt T) bool {
	var zero T
	return opt != zero
}

// SendMessage holds all configurable options for the send-message Telegram
// API call. Fields that map to optional API flags default to their zero
// values; the caller only needs to set the ones they care about.
//
// Example:
//
//	opt := &params.SendMessage{
//	    DisableNotification: true,
//	    ReplyToMessageID:    42,
//	    ParseMode:           params.ParseModeHTML,
//	}
//	msg, err := client.SendMessage(ctx, chatID, "<b>Hello</b>", opt)
type SendMessage struct {
	DisableWebPagePreview bool
	DisableNotification   bool
	Silent                bool
	Background            bool
	ClearDraft            bool
	NoForwards            bool
	InvertMedia           bool
	ReplyToMessageID      int32
	ReplyTo               tl.InputReplyToClass
	ReplyMarkup           tl.ReplyMarkupClass
	Entities              []tl.MessageEntityClass
	ParseMode             ParseMode
	ScheduleDate          *int32
	EffectID              *int64
	SendAs                tl.InputPeerClass
	RepeatPeriod          *int32
	BusinessConnectionID  string
	AllowPaidBroadcast    bool
	PaidMessageStarCount  *int64
	ShowCaptionAboveMedia bool
	MessageThreadID       int32
	DirectMessagesTopicID int64
	ProtectContent        bool
}

// EditMessage holds all configurable options for the edit-message Telegram
// API call. Only the fields that are set will be applied as changes.
//
// Example:
//
//	opt := &params.EditMessage{
//	    ParseMode: params.ParseModeHTML,
//	}
//	edited, err := msg.Edit("<i>Updated</i>", opt)
type EditMessage struct {
	DisableWebPagePreview bool
	InvertMedia           bool
	ReplyMarkup           tl.ReplyMarkupClass
	ParseMode             ParseMode
	Entities              []tl.MessageEntityClass
	ScheduleDate          *int32
	ShowCaptionAboveMedia bool
	BusinessConnectionID  string
}

// ForwardMessages holds all configurable options for the forward-messages
// Telegram API call.
//
// Example:
//
//	opt := &params.ForwardMessages{
//	    DisableNotification: true,
//	    DropAuthor:          true,
//	}
//	fwd, err := msg.Forward(targetChatID, opt)
type ForwardMessages struct {
	DisableNotification  bool
	NoForwards           bool
	DropAuthor           bool
	DropMediaCaptions    bool
	ScheduleDate         *int32
	RepeatPeriod         *int32
	HideSenderName       bool
	HideCaptions         bool
	AllowPaidBroadcast   bool
	PaidMessageStarCount *int64
	MessageThreadID      int32
	ProtectContent       bool
}

// CopyMessage holds all configurable options for the copy-message Telegram
// API call. Unlike forwarding, copying creates a new message with identical
// content but without the original author link.
//
// Example:
//
//	opt := &params.CopyMessage{
//	    Caption:             "Re-shared photo",
//	    DisableNotification: true,
//	}
//	newID, err := msg.Copy(targetChatID, opt)
type CopyMessage struct {
	Caption               string
	ParseMode             ParseMode
	CaptionEntities       []tl.MessageEntityClass
	DisableNotification   bool
	ReplyToMessageID      int32
	ReplyMarkup           tl.ReplyMarkupClass
	ScheduleDate          *int32
	DropAuthor            bool
	HasSpoiler            bool
	ShowCaptionAboveMedia bool
	BusinessConnectionID  string
	AllowPaidBroadcast    bool
	PaidMessageStarCount  *int64
	MessageThreadID       int32
	ProtectContent        bool
	NoForwards            bool
}

// DeleteMessages holds options for the delete-messages Telegram API call.
type DeleteMessages struct {
	// Revoke deletes the messages for all chat participants, not just the
	// caller. When false, messages are deleted locally only.
	Revoke bool
}

// PinMessage holds options for the pin-message Telegram API call.
type PinMessage struct {
	// Silent pins the message without triggering a notification in the
	// chat.
	Silent bool

	// Unpin unpins the message instead of pinning it. Use this as a
	// convenience flag to unpin without calling a separate API method.
	Unpin bool
}

// ProgressInfo describes the current state of a file upload or download
// operation. Handlers receive this struct on each progress update to report
// throughput or render progress bars.
//
// Example:
//
//	progressCb := func(info params.ProgressInfo) {
//	    pct := info.Progress()
//	    fmt.Printf("\r%.1f%% complete (%d / %d bytes)", pct, info.UploadedBytes, info.TotalBytes)
//	}
type ProgressInfo struct {
	// FileName is the name of the file being transferred.
	FileName string

	// TotalBytes is the expected total size of the file in bytes. May be 0
	// if the server does not report a Content-Length.
	TotalBytes int64

	// UploadedBytes is the number of bytes sent so far during an upload.
	UploadedBytes int64

	// DownloadedBytes is the number of bytes received so far during a
	// download.
	DownloadedBytes int64

	// IsUpload is true when the operation is an upload, false for a
	// download. Determines whether UploadedBytes or DownloadedBytes is
	// used in the progress calculation.
	IsUpload bool
}

// Progress returns the transfer completion percentage as a float64 in the
// range [0, 100]. Returns 0 when TotalBytes is 0 (unknown size).
//
// Example:
//
//	info := params.ProgressInfo{TotalBytes: 1000, UploadedBytes: 250, IsUpload: true}
//	fmt.Printf("%.0f%% done\n", info.Progress()) // 25% done
func (p ProgressInfo) Progress() float64 {
	if p.TotalBytes == 0 {
		return 0
	}
	var done int64
	if p.IsUpload {
		done = p.UploadedBytes
	} else {
		done = p.DownloadedBytes
	}
	return float64(done) / float64(p.TotalBytes) * 100
}

// ProgressFunc is the callback signature for file-transfer progress updates.
// The receiver is invoked periodically during uploads and downloads so the
// caller can report throughput, update progress bars, or cancel long-running
// transfers.
type ProgressFunc func(info ProgressInfo)

// Download holds configurable options for file-download Telegram API calls.
//
// Example:
//
//	data, err := msg.Download(&params.Download{
//	    ChunkSize: 512 * 1024,
//	    Progress:  func(info params.ProgressInfo) {
//	        fmt.Printf("Downloaded %.1f%%\n", info.Progress())
//	    },
//	})
type Download struct {
	// FileName is the target file path for file-based downloads.
	FileName string

	// ChunkSize is the maximum number of bytes to request per chunk. Larger
	// values reduce round-trips but increase memory usage. A value of 0
	// lets the client choose a sensible default.
	ChunkSize int32

	// Progress is an optional callback invoked on each chunk transfer to
	// report download progress. Nil disables progress reporting.
	Progress ProgressFunc

	// DCID specifies the data-center ID to download from. A value of 0
	// lets the client automatically resolve the correct data center.
	DCID int32
}

// GetGifts holds options for the payments.getSavedStarGifts API call.
// Fields map to optional filter flags on the request.
type GetGifts struct {
	ExcludeUnsaved      bool
	ExcludeSaved        bool
	ExcludeUnlimited    bool
	ExcludeUnique       bool
	SortByValue         bool
	ExcludeUpgradable   bool
	ExcludeUnupgradable bool
	PeerColorAvailable  bool
	ExcludeHosted       bool
	CollectionID        int32
	Offset              string
	Limit               int32
}

// SendPoll holds all configurable options for the send-poll Telegram API call.
// Fields that map to optional API flags default to their zero values; the
// caller only needs to set the ones they care about.
//
// Example:
//
//	opt := &params.SendPoll{
//	    DisableNotification: true,
//	    MultipleChoice:      true,
//	    ClosePeriod:         ptr.Int32(300),
//	}
//	msg, err := client.SendPoll(ctx, chatID, "Pick one", []string{"A", "B", "C"}, opt)
type SendPoll struct {
	DisableNotification   bool
	Silent                bool
	Background            bool
	ClearDraft            bool
	NoForwards            bool
	ProtectContent        bool
	ReplyToMessageID      int32
	ReplyTo               tl.InputReplyToClass
	ReplyMarkup           tl.ReplyMarkupClass
	ScheduleDate          *int32
	EffectID              *int64
	SendAs                tl.InputPeerClass
	PublicVoters          bool
	MultipleChoice        bool
	Quiz                  bool
	Closed                bool
	ShuffleAnswers        bool
	RevotingDisabled      bool
	HideResultsUntilClose bool
	SubscribersOnly       bool
	OpenAnswers           bool
	AllowAddingOptions    bool
	ClosePeriod           *int32
	CloseDate             *int32
	CorrectAnswers        [][]byte
	Solution              *string
	SolutionEntities      []tl.MessageEntityClass
	Description           string
	DescriptionMedia      tl.InputMediaClass
	BusinessConnectionID  string
	AllowPaidBroadcast    bool
	PaidMessageStarCount  *int64
	MessageThreadID       int32
	RepeatPeriod          *int32
}

// SendVenue holds all configurable options for the send-venue Telegram API call.
// Fields that map to optional API flags default to their zero values; the
// caller only needs to set the ones they care about.
//
// Example:
//
//	opt := &params.SendVenue{
//	    DisableNotification: true,
//	    FoursquareID:        "4b8e3f56f964a520c3e432e3",
//	}
//	msg, err := client.SendVenue(ctx, chatID, 40.7128, -74.0060, "HQ", "123 Main St", opt)
type SendVenue struct {
	DisableNotification  bool
	Silent               bool
	Background           bool
	ClearDraft           bool
	NoForwards           bool
	ProtectContent       bool
	ReplyToMessageID     int32
	ReplyTo              tl.InputReplyToClass
	ReplyMarkup          tl.ReplyMarkupClass
	ScheduleDate         *int32
	EffectID             *int64
	SendAs               tl.InputPeerClass
	Provider             string
	VenueID              string
	VenueType            string
	FoursquareID         string
	FoursquareType       string
	BusinessConnectionID string
	AllowPaidBroadcast   bool
	PaidMessageStarCount *int64
	MessageThreadID      int32
}

// SendContact holds all configurable options for the send-contact Telegram API
// call. Fields that map to optional API flags default to their zero values; the
// caller only needs to set the ones they care about.
//
// Example:
//
//	opt := &params.SendContact{
//	    DisableNotification: true,
//	    Vcard:               "BEGIN:VCARD\nVERSION:3.0\nFN:Alice\nEND:VCARD",
//	}
//	msg, err := client.SendContact(ctx, chatID, "Alice", "Smith", "15551234567", opt)
type SendContact struct {
	DisableNotification  bool
	Silent               bool
	Background           bool
	ClearDraft           bool
	NoForwards           bool
	ProtectContent       bool
	ReplyToMessageID     int32
	ReplyTo              tl.InputReplyToClass
	ReplyMarkup          tl.ReplyMarkupClass
	ScheduleDate         *int32
	EffectID             *int64
	SendAs               tl.InputPeerClass
	Vcard                string
	BusinessConnectionID string
	AllowPaidBroadcast   bool
	PaidMessageStarCount *int64
	MessageThreadID      int32
}

// SendLocation holds all configurable options for the send-location Telegram
// API call. Supports live locations via LivePeriod and heading/proximity
// alerts. Fields that map to optional API flags default to their zero values.
//
// Example:
//
//	opt := &params.SendLocation{
//	    LivePeriod:  ptr.Int32(3600),
//	    Heading:     ptr.Int32(90),
//	}
//	msg, err := client.SendLocation(ctx, chatID, 40.7128, -74.0060, opt)
type SendLocation struct {
	DisableNotification  bool
	Silent               bool
	Background           bool
	ClearDraft           bool
	NoForwards           bool
	ProtectContent       bool
	ReplyToMessageID     int32
	ReplyTo              tl.InputReplyToClass
	ReplyMarkup          tl.ReplyMarkupClass
	ScheduleDate         *int32
	EffectID             *int64
	SendAs               tl.InputPeerClass
	AccuracyRadius       *int32
	Heading              *int32
	ProximityAlertRadius *int32
	LivePeriod           *int32
	BusinessConnectionID string
	AllowPaidBroadcast   bool
	PaidMessageStarCount *int64
	MessageThreadID      int32
}

// SendDice holds all configurable options for the send-dice Telegram API call.
// The Emoticon field selects the dice type (e.g. "🎲", "🎯", "🏀"). Fields
// that map to optional API flags default to their zero values.
//
// Example:
//
//	opt := &params.SendDice{
//	    Emoticon: "🎲",
//	}
//	msg, err := client.SendDice(ctx, chatID, opt)
type SendDice struct {
	DisableNotification  bool
	Silent               bool
	Background           bool
	ClearDraft           bool
	NoForwards           bool
	ProtectContent       bool
	ReplyToMessageID     int32
	ReplyTo              tl.InputReplyToClass
	ReplyMarkup          tl.ReplyMarkupClass
	ScheduleDate         *int32
	EffectID             *int64
	SendAs               tl.InputPeerClass
	Emoticon             string
	BusinessConnectionID string
	AllowPaidBroadcast   bool
	PaidMessageStarCount *int64
	MessageThreadID      int32
}

// SendGame holds all configurable options for the send-game Telegram API call.
// Fields that map to optional API flags default to their zero values; the
// caller only needs to set the ones they care about.
//
// Example:
//
//	opt := &params.SendGame{
//	    DisableNotification: true,
//	}
//	msg, err := client.SendGame(ctx, chatID, "my_short_name", opt)
type SendGame struct {
	DisableNotification  bool
	Silent               bool
	Background           bool
	ClearDraft           bool
	NoForwards           bool
	ProtectContent       bool
	ReplyToMessageID     int32
	ReplyTo              tl.InputReplyToClass
	ReplyMarkup          tl.ReplyMarkupClass
	ScheduleDate         *int32
	EffectID             *int64
	SendAs               tl.InputPeerClass
	BusinessConnectionID string
	AllowPaidBroadcast   bool
	PaidMessageStarCount *int64
	MessageThreadID      int32
}

// SendMediaGroup holds all configurable options for the send-media-group
// Telegram API call, used to send an album of photos, videos, or documents as
// a single grouped message. Fields that map to optional API flags default to
// their zero values.
//
// Example:
//
//	opt := &params.SendMediaGroup{
//	    DisableNotification: true,
//	}
//	msgs, err := client.SendMediaGroup(ctx, chatID, mediaItems, opt)
type SendMediaGroup struct {
	DisableNotification   bool
	Silent                bool
	Background            bool
	ClearDraft            bool
	NoForwards            bool
	ProtectContent        bool
	ReplyToMessageID      int32
	ReplyTo               tl.InputReplyToClass
	ScheduleDate          *int32
	EffectID              *int64
	SendAs                tl.InputPeerClass
	ShowCaptionAboveMedia bool
	BusinessConnectionID  string
	AllowPaidBroadcast    bool
	PaidMessageStarCount  *int64
	MessageThreadID       int32
}

// SendChecklist holds all configurable options for the send-checklist Telegram
// API call. Checklists are interactive to-do lists with tasks that users can
// complete. Fields that map to optional API flags default to their zero values.
//
// Example:
//
//	opt := &params.SendChecklist{
//	    OthersCanAppend:   true,
//	    OthersCanComplete: true,
//	}
//	msg, err := client.SendChecklist(ctx, chatID, "Shopping list", tasks, opt)
type SendChecklist struct {
	DisableNotification  bool
	Silent               bool
	Background           bool
	ClearDraft           bool
	NoForwards           bool
	ProtectContent       bool
	ReplyToMessageID     int32
	ReplyTo              tl.InputReplyToClass
	ReplyMarkup          tl.ReplyMarkupClass
	ScheduleDate         *int32
	EffectID             *int64
	SendAs               tl.InputPeerClass
	OthersCanAppend      bool
	OthersCanComplete    bool
	RepeatPeriod         *int32
	PaidMessageStars     *int64
	BusinessConnectionID string
	MessageThreadID      int32
}

func (s *SendChecklist) getFlatSendFields() (bool, bool, bool, bool, bool, int32, tl.InputReplyToClass, tl.ReplyMarkupClass, *int32, *int64, tl.InputPeerClass) {
	return s.DisableNotification, s.Silent, s.Background, s.ClearDraft, s.NoForwards, s.ReplyToMessageID, s.ReplyTo, s.ReplyMarkup, s.ScheduleDate, s.EffectID, s.SendAs
}
func (s *SendChecklist) ToSendMsg() *SendMessage { return flatToSendMsg(s) }

// SendInlineBotResult holds all configurable options for sending an inline bot
// result to a chat. Fields that map to optional API flags default to their
// zero values; the caller only needs to set the ones they care about.
//
// Example:
//
//	opt := &params.SendInlineBotResult{
//	    DisableNotification: true,
//	    HideVia:             true,
//	}
//	msg, err := client.SendInlineBotResult(ctx, chatID, queryID, resultID, opt)
type SendInlineBotResult struct {
	DisableNotification  bool
	Silent               bool
	Background           bool
	ClearDraft           bool
	NoForwards           bool
	ProtectContent       bool
	ReplyToMessageID     int32
	ReplyTo              tl.InputReplyToClass
	ReplyMarkup          tl.ReplyMarkupClass
	ScheduleDate         *int32
	EffectID             *int64
	SendAs               tl.InputPeerClass
	HideVia              bool
	AllowPaidStars       *int64
	BusinessConnectionID string
	AllowPaidBroadcast   bool
	PaidMessageStarCount *int64
	MessageThreadID      int32
}

func flatToSendMsg(s interface {
	getFlatSendFields() (bool, bool, bool, bool, bool, int32, tl.InputReplyToClass, tl.ReplyMarkupClass, *int32, *int64, tl.InputPeerClass)
}) *SendMessage {
	disableNotification, silent, background, clearDraft, noForwards,
		replyToMessageID, replyTo, replyMarkup, scheduleDate, effectID, sendAs := s.getFlatSendFields()
	return &SendMessage{
		DisableNotification: disableNotification,
		Silent:              silent,
		Background:          background,
		ClearDraft:          clearDraft,
		NoForwards:          noForwards,
		ReplyToMessageID:    replyToMessageID,
		ReplyTo:             replyTo,
		ReplyMarkup:         replyMarkup,
		ScheduleDate:        scheduleDate,
		EffectID:            effectID,
		SendAs:              sendAs,
	}
}

func (s *SendPoll) getFlatSendFields() (bool, bool, bool, bool, bool, int32, tl.InputReplyToClass, tl.ReplyMarkupClass, *int32, *int64, tl.InputPeerClass) {
	return s.DisableNotification, s.Silent, s.Background, s.ClearDraft, s.NoForwards, s.ReplyToMessageID, s.ReplyTo, s.ReplyMarkup, s.ScheduleDate, s.EffectID, s.SendAs
}
func (s *SendPoll) ToSendMsg() *SendMessage { return flatToSendMsg(s) }

func (s *SendVenue) getFlatSendFields() (bool, bool, bool, bool, bool, int32, tl.InputReplyToClass, tl.ReplyMarkupClass, *int32, *int64, tl.InputPeerClass) {
	return s.DisableNotification, s.Silent, s.Background, s.ClearDraft, s.NoForwards, s.ReplyToMessageID, s.ReplyTo, s.ReplyMarkup, s.ScheduleDate, s.EffectID, s.SendAs
}
func (s *SendVenue) ToSendMsg() *SendMessage { return flatToSendMsg(s) }

func (s *SendContact) getFlatSendFields() (bool, bool, bool, bool, bool, int32, tl.InputReplyToClass, tl.ReplyMarkupClass, *int32, *int64, tl.InputPeerClass) {
	return s.DisableNotification, s.Silent, s.Background, s.ClearDraft, s.NoForwards, s.ReplyToMessageID, s.ReplyTo, s.ReplyMarkup, s.ScheduleDate, s.EffectID, s.SendAs
}
func (s *SendContact) ToSendMsg() *SendMessage { return flatToSendMsg(s) }

func (s *SendLocation) getFlatSendFields() (bool, bool, bool, bool, bool, int32, tl.InputReplyToClass, tl.ReplyMarkupClass, *int32, *int64, tl.InputPeerClass) {
	return s.DisableNotification, s.Silent, s.Background, s.ClearDraft, s.NoForwards, s.ReplyToMessageID, s.ReplyTo, s.ReplyMarkup, s.ScheduleDate, s.EffectID, s.SendAs
}
func (s *SendLocation) ToSendMsg() *SendMessage { return flatToSendMsg(s) }

func (s *SendDice) getFlatSendFields() (bool, bool, bool, bool, bool, int32, tl.InputReplyToClass, tl.ReplyMarkupClass, *int32, *int64, tl.InputPeerClass) {
	return s.DisableNotification, s.Silent, s.Background, s.ClearDraft, s.NoForwards, s.ReplyToMessageID, s.ReplyTo, s.ReplyMarkup, s.ScheduleDate, s.EffectID, s.SendAs
}
func (s *SendDice) ToSendMsg() *SendMessage { return flatToSendMsg(s) }

func (s *SendGame) getFlatSendFields() (bool, bool, bool, bool, bool, int32, tl.InputReplyToClass, tl.ReplyMarkupClass, *int32, *int64, tl.InputPeerClass) {
	return s.DisableNotification, s.Silent, s.Background, s.ClearDraft, s.NoForwards, s.ReplyToMessageID, s.ReplyTo, s.ReplyMarkup, s.ScheduleDate, s.EffectID, s.SendAs
}
func (s *SendGame) ToSendMsg() *SendMessage { return flatToSendMsg(s) }

func (s *SendMediaGroup) getFlatSendFields() (bool, bool, bool, bool, bool, int32, tl.InputReplyToClass, tl.ReplyMarkupClass, *int32, *int64, tl.InputPeerClass) {
	return s.DisableNotification, s.Silent, s.Background, s.ClearDraft, s.NoForwards, s.ReplyToMessageID, s.ReplyTo, nil, s.ScheduleDate, s.EffectID, s.SendAs
}
func (s *SendMediaGroup) ToSendMsg() *SendMessage { return flatToSendMsg(s) }

func (s *SendInlineBotResult) getFlatSendFields() (bool, bool, bool, bool, bool, int32, tl.InputReplyToClass, tl.ReplyMarkupClass, *int32, *int64, tl.InputPeerClass) {
	return s.DisableNotification, s.Silent, s.Background, s.ClearDraft, s.NoForwards, s.ReplyToMessageID, s.ReplyTo, s.ReplyMarkup, s.ScheduleDate, s.EffectID, s.SendAs
}
func (s *SendInlineBotResult) ToSendMsg() *SendMessage { return flatToSendMsg(s) }

// InlineQuery holds options for answering an inline query via
// messages.setInlineBotResults.
type InlineQuery struct {
	// CacheTime is the number of seconds the client should cache the results.
	CacheTime int
	// Gallery enables gallery-style result layout.
	Gallery bool
	// Private marks the results as private (only visible to the querying user).
	Private bool
	// NextOffset is the pagination offset for the next batch of results.
	NextOffset string
	// SwitchPM is the start parameter for a switch-to-PM button.
	SwitchPM string
	// SwitchPMText is the label shown on the switch-to-PM button.
	SwitchPMText string
}

// SendAudio holds all configurable options for the send-audio Telegram API call.
// Audio files are displayed as music players in the chat. Fields that map to
// optional API flags default to their zero values.
//
// Example:
//
//	opt := &params.SendAudio{
//	    Performer:           "Daft Punk",
//	    Title:               "Around the World",
//	    ParseMode:           params.ParseModeHTML,
//	    ShowCaptionAboveMedia: true,
//	}
//	msg, err := client.SendAudio(ctx, chatID, file, "<b>New track!</b>", opt)
type SendAudio struct {
	DisableNotification   bool
	Silent                bool
	Background            bool
	ClearDraft            bool
	NoForwards            bool
	ProtectContent        bool
	ReplyToMessageID      int32
	ReplyTo               tl.InputReplyToClass
	ReplyMarkup           tl.ReplyMarkupClass
	ScheduleDate          *int32
	EffectID              *int64
	SendAs                tl.InputPeerClass
	Duration              int32
	Performer             string
	Title                 string
	FileName              string
	Thumb                 string
	ParseMode             ParseMode
	CaptionEntities       []tl.MessageEntityClass
	ShowCaptionAboveMedia bool
	BusinessConnectionID  string
	AllowPaidBroadcast    bool
	PaidMessageStarCount  *int64
	MessageThreadID       int32
	RepeatPeriod          *int32
}

func (s *SendAudio) getFlatSendFields() (bool, bool, bool, bool, bool, int32, tl.InputReplyToClass, tl.ReplyMarkupClass, *int32, *int64, tl.InputPeerClass) {
	return s.DisableNotification, s.Silent, s.Background, s.ClearDraft, s.NoForwards, s.ReplyToMessageID, s.ReplyTo, s.ReplyMarkup, s.ScheduleDate, s.EffectID, s.SendAs
}
func (s *SendAudio) ToSendMsg() *SendMessage { return flatToSendMsg(s) }

// SendVideo holds all configurable options for the send-video Telegram API
// call. Video files are streamed directly in the chat. Fields that map to
// optional API flags default to their zero values.
//
// Example:
//
//	opt := &params.SendVideo{
//	    Duration:            120.5,
//	    Width:               1920,
//	    Height:              1080,
//	    SupportsStreaming:   true,
//	    ParseMode:           params.ParseModeHTML,
//	}
//	msg, err := client.SendVideo(ctx, chatID, file, "<b>Check this out</b>", opt)
type SendVideo struct {
	DisableNotification   bool
	Silent                bool
	Background            bool
	ClearDraft            bool
	NoForwards            bool
	ProtectContent        bool
	ReplyToMessageID      int32
	ReplyTo               tl.InputReplyToClass
	ReplyMarkup           tl.ReplyMarkupClass
	ScheduleDate          *int32
	EffectID              *int64
	SendAs                tl.InputPeerClass
	Duration              float64
	Width                 int32
	Height                int32
	SupportsStreaming     bool
	FileName              string
	Thumb                 string
	ParseMode             ParseMode
	CaptionEntities       []tl.MessageEntityClass
	HasSpoiler            bool
	TTLSeconds            *int32
	ViewOnce              bool
	ShowCaptionAboveMedia bool
	BusinessConnectionID  string
	AllowPaidBroadcast    bool
	PaidMessageStarCount  *int64
	MessageThreadID       int32
	RepeatPeriod          *int32
	NoSound               bool
	VideoStartTimestamp   *int32
	VideoCover            tl.InputDocumentClass
}

func (s *SendVideo) getFlatSendFields() (bool, bool, bool, bool, bool, int32, tl.InputReplyToClass, tl.ReplyMarkupClass, *int32, *int64, tl.InputPeerClass) {
	return s.DisableNotification, s.Silent, s.Background, s.ClearDraft, s.NoForwards, s.ReplyToMessageID, s.ReplyTo, s.ReplyMarkup, s.ScheduleDate, s.EffectID, s.SendAs
}
func (s *SendVideo) ToSendMsg() *SendMessage { return flatToSendMsg(s) }

// SendDocument holds all configurable options for the send-document Telegram
// API call. Documents are general-purpose file attachments. Fields that map to
// optional API flags default to their zero values.
//
// Example:
//
//	opt := &params.SendDocument{
//	    FileName:     "report.pdf",
//	    ForceDocument: true,
//	    ParseMode:    params.ParseModeHTML,
//	}
//	msg, err := client.SendDocument(ctx, chatID, file, "Q4 <b>report</b>", opt)
type SendDocument struct {
	DisableNotification   bool
	Silent                bool
	Background            bool
	ClearDraft            bool
	NoForwards            bool
	ProtectContent        bool
	ReplyToMessageID      int32
	ReplyTo               tl.InputReplyToClass
	ReplyMarkup           tl.ReplyMarkupClass
	ScheduleDate          *int32
	EffectID              *int64
	SendAs                tl.InputPeerClass
	FileName              string
	Thumb                 string
	MimeType              string
	ForceDocument         bool
	ParseMode             ParseMode
	CaptionEntities       []tl.MessageEntityClass
	ShowCaptionAboveMedia bool
	BusinessConnectionID  string
	AllowPaidBroadcast    bool
	PaidMessageStarCount  *int64
	MessageThreadID       int32
	RepeatPeriod          *int32
}

func (s *SendDocument) getFlatSendFields() (bool, bool, bool, bool, bool, int32, tl.InputReplyToClass, tl.ReplyMarkupClass, *int32, *int64, tl.InputPeerClass) {
	return s.DisableNotification, s.Silent, s.Background, s.ClearDraft, s.NoForwards, s.ReplyToMessageID, s.ReplyTo, s.ReplyMarkup, s.ScheduleDate, s.EffectID, s.SendAs
}
func (s *SendDocument) ToSendMsg() *SendMessage { return flatToSendMsg(s) }

// SendPhoto holds all configurable options for the send-photo Telegram API
// call. Photos are displayed inline in the chat with optional captions. Fields
// that map to optional API flags default to their zero values.
//
// Example:
//
//	opt := &params.SendPhoto{
//	    HasSpoiler:  true,
//	    ParseMode:   params.ParseModeHTML,
//	    TTLSeconds:  ptr.Int32(60),
//	}
//	msg, err := client.SendPhoto(ctx, chatID, file, "<i>Disappearing</i>", opt)
type SendPhoto struct {
	DisableNotification   bool
	Silent                bool
	Background            bool
	ClearDraft            bool
	NoForwards            bool
	ProtectContent        bool
	ReplyToMessageID      int32
	ReplyTo               tl.InputReplyToClass
	ReplyMarkup           tl.ReplyMarkupClass
	ScheduleDate          *int32
	EffectID              *int64
	SendAs                tl.InputPeerClass
	FileName              string
	ParseMode             ParseMode
	CaptionEntities       []tl.MessageEntityClass
	HasSpoiler            bool
	TTLSeconds            *int32
	ViewOnce              bool
	ShowCaptionAboveMedia bool
	BusinessConnectionID  string
	AllowPaidBroadcast    bool
	PaidMessageStarCount  *int64
	MessageThreadID       int32
	RepeatPeriod          *int32
}

func (s *SendPhoto) getFlatSendFields() (bool, bool, bool, bool, bool, int32, tl.InputReplyToClass, tl.ReplyMarkupClass, *int32, *int64, tl.InputPeerClass) {
	return s.DisableNotification, s.Silent, s.Background, s.ClearDraft, s.NoForwards, s.ReplyToMessageID, s.ReplyTo, s.ReplyMarkup, s.ScheduleDate, s.EffectID, s.SendAs
}
func (s *SendPhoto) ToSendMsg() *SendMessage { return flatToSendMsg(s) }

// SendAnimation holds all configurable options for the send-animation Telegram
// API call. Animations are square-cropped video-like content (GIFs or MP4s
// without sound). Fields that map to optional API flags default to their zero
// values.
//
// Example:
//
//	opt := &params.SendAnimation{
//	    Width:       320,
//	    Height:      240,
//	    HasSpoiler:  true,
//	    ParseMode:   params.ParseModeHTML,
//	}
//	msg, err := client.SendAnimation(ctx, chatID, file, "<b>funny</b>", opt)
type SendAnimation struct {
	DisableNotification   bool
	Silent                bool
	Background            bool
	ClearDraft            bool
	NoForwards            bool
	ProtectContent        bool
	ReplyToMessageID      int32
	ReplyTo               tl.InputReplyToClass
	ReplyMarkup           tl.ReplyMarkupClass
	ScheduleDate          *int32
	EffectID              *int64
	SendAs                tl.InputPeerClass
	FileName              string
	Thumb                 string
	ParseMode             ParseMode
	CaptionEntities       []tl.MessageEntityClass
	HasSpoiler            bool
	Duration              float64
	Width                 int32
	Height                int32
	Unsave                bool
	ShowCaptionAboveMedia bool
	BusinessConnectionID  string
	AllowPaidBroadcast    bool
	PaidMessageStarCount  *int64
	MessageThreadID       int32
	RepeatPeriod          *int32
}

func (s *SendAnimation) getFlatSendFields() (bool, bool, bool, bool, bool, int32, tl.InputReplyToClass, tl.ReplyMarkupClass, *int32, *int64, tl.InputPeerClass) {
	return s.DisableNotification, s.Silent, s.Background, s.ClearDraft, s.NoForwards, s.ReplyToMessageID, s.ReplyTo, s.ReplyMarkup, s.ScheduleDate, s.EffectID, s.SendAs
}
func (s *SendAnimation) ToSendMsg() *SendMessage { return flatToSendMsg(s) }

// SendVoice holds all configurable options for the send-voice Telegram API call.
// Voice messages are displayed as waveform audio players in the chat. Fields
// that map to optional API flags default to their zero values.
//
// Example:
//
//	opt := &params.SendVoice{
//	    Duration:  15,
//	    ParseMode: params.ParseModeHTML,
//	}
//	msg, err := client.SendVoice(ctx, chatID, file, "<b>Voice note</b>", opt)
type SendVoice struct {
	DisableNotification   bool
	Silent                bool
	Background            bool
	ClearDraft            bool
	NoForwards            bool
	ProtectContent        bool
	ReplyToMessageID      int32
	ReplyTo               tl.InputReplyToClass
	ReplyMarkup           tl.ReplyMarkupClass
	ScheduleDate          *int32
	EffectID              *int64
	SendAs                tl.InputPeerClass
	Duration              int32
	FileName              string
	ParseMode             ParseMode
	CaptionEntities       []tl.MessageEntityClass
	ViewOnce              bool
	ShowCaptionAboveMedia bool
	BusinessConnectionID  string
	AllowPaidBroadcast    bool
	PaidMessageStarCount  *int64
	MessageThreadID       int32
	RepeatPeriod          *int32
}

func (s *SendVoice) getFlatSendFields() (bool, bool, bool, bool, bool, int32, tl.InputReplyToClass, tl.ReplyMarkupClass, *int32, *int64, tl.InputPeerClass) {
	return s.DisableNotification, s.Silent, s.Background, s.ClearDraft, s.NoForwards, s.ReplyToMessageID, s.ReplyTo, s.ReplyMarkup, s.ScheduleDate, s.EffectID, s.SendAs
}
func (s *SendVoice) ToSendMsg() *SendMessage { return flatToSendMsg(s) }

// SendVideoNote holds all configurable options for the send-video-note Telegram
// API call. Video notes are round-formatted video messages displayed inline in
// the chat. Fields that map to optional API flags default to their zero values.
//
// Example:
//
//	opt := &params.SendVideoNote{
//	    Duration: 10.0,
//	    Length:   240,
//	    ViewOnce: true,
//	}
//	msg, err := client.SendVideoNote(ctx, chatID, file, opt)
type SendVideoNote struct {
	DisableNotification  bool
	Silent               bool
	Background           bool
	ClearDraft           bool
	NoForwards           bool
	ProtectContent       bool
	ReplyToMessageID     int32
	ReplyTo              tl.InputReplyToClass
	ReplyMarkup          tl.ReplyMarkupClass
	ScheduleDate         *int32
	EffectID             *int64
	SendAs               tl.InputPeerClass
	Duration             float64
	FileName             string
	Thumb                string
	Length               int32
	ViewOnce             bool
	BusinessConnectionID string
	AllowPaidBroadcast   bool
	PaidMessageStarCount *int64
	MessageThreadID      int32
	RepeatPeriod         *int32
}

func (s *SendVideoNote) getFlatSendFields() (bool, bool, bool, bool, bool, int32, tl.InputReplyToClass, tl.ReplyMarkupClass, *int32, *int64, tl.InputPeerClass) {
	return s.DisableNotification, s.Silent, s.Background, s.ClearDraft, s.NoForwards, s.ReplyToMessageID, s.ReplyTo, s.ReplyMarkup, s.ScheduleDate, s.EffectID, s.SendAs
}
func (s *SendVideoNote) ToSendMsg() *SendMessage { return flatToSendMsg(s) }

// SendSticker holds all configurable options for the send-sticker Telegram API
// call. Stickers are special image objects displayed at a fixed size with
// optional associated emoji. Fields that map to optional API flags default to
// their zero values.
//
// Example:
//
//	opt := &params.SendSticker{
//	    Emoji: "🎉",
//	}
//	msg, err := client.SendSticker(ctx, chatID, stickerFile, opt)
type SendSticker struct {
	DisableNotification  bool
	Silent               bool
	Background           bool
	ClearDraft           bool
	NoForwards           bool
	ProtectContent       bool
	ReplyToMessageID     int32
	ReplyTo              tl.InputReplyToClass
	ReplyMarkup          tl.ReplyMarkupClass
	ScheduleDate         *int32
	EffectID             *int64
	SendAs               tl.InputPeerClass
	FileName             string
	Emoji                string
	ParseMode            ParseMode
	CaptionEntities      []tl.MessageEntityClass
	BusinessConnectionID string
	AllowPaidBroadcast   bool
	PaidMessageStarCount *int64
	MessageThreadID      int32
	RepeatPeriod         *int32
}

func (s *SendSticker) getFlatSendFields() (bool, bool, bool, bool, bool, int32, tl.InputReplyToClass, tl.ReplyMarkupClass, *int32, *int64, tl.InputPeerClass) {
	return s.DisableNotification, s.Silent, s.Background, s.ClearDraft, s.NoForwards, s.ReplyToMessageID, s.ReplyTo, s.ReplyMarkup, s.ScheduleDate, s.EffectID, s.SendAs
}
func (s *SendSticker) ToSendMsg() *SendMessage { return flatToSendMsg(s) }

// AnswerCallback holds options for answering a callback query from an inline
// button press. Text is shown as a toast notification; ShowAlert promotes it
// to a full alert dialog.
//
// Example:
//
//	opt := &params.AnswerCallback{
//	    Text:      "Saved!",
//	    ShowAlert: false,
//	}
//	err := callback.Answer(opt)
type AnswerCallback struct {
	Text      string
	ShowAlert bool
	URL       string
	CacheTime int32
}

// AnswerShipping holds options for answering a shipping query in a payment
// flow. Set Ok to true with ShippingOptions to accept, or false with ErrorMsg
// to reject.
//
// Example:
//
//	opt := &params.AnswerShipping{
//	    Ok:              true,
//	    ShippingOptions: shippingOpts,
//	}
//	err := shippingQuery.Answer(opt)
type AnswerShipping struct {
	Ok              bool
	ShippingOptions interface{}
	ErrorMsg        string
}

// AnswerPreCheckout holds options for answering a pre-checkout query in a
// payment flow. Set Ok to true to confirm, or false with ErrorMsg to decline.
//
// Example:
//
//	opt := &params.AnswerPreCheckout{
//	    Ok: true,
//	}
//	err := preCheckoutQuery.Answer(opt)
type AnswerPreCheckout struct {
	Ok       bool
	ErrorMsg string
}

// EditCaption holds options for editing the caption of a previously sent
// media message. Fields that map to optional API flags default to their zero
// values.
//
// Example:
//
//	opt := &params.EditCaption{
//	    Caption:   "Updated caption",
//	    ParseMode: params.ParseModeHTML,
//	}
//	err := msg.EditCaption("<b>New caption</b>", opt)
type EditCaption struct {
	Caption               string
	ParseMode             ParseMode
	CaptionEntities       []tl.MessageEntityClass
	ReplyMarkup           tl.ReplyMarkupClass
	ShowCaptionAboveMedia bool
	BusinessConnectionID  string
	ScheduleDate          *int32
}

// StoryForward holds options for forwarding a story to a chat. Fields that map
// to optional API flags default to their zero values.
//
// Example:
//
//	opt := &params.StoryForward{
//	    DisableNotification: true,
//	}
//	err := client.ForwardStory(ctx, chatID, fromPeer, storyID, opt)
type StoryForward struct {
	DisableNotification  bool
	MessageThreadID      int32
	ScheduleDate         *int32
	RepeatPeriod         *int32
	PaidMessageStarCount *int64
	ProtectContent       bool
	AllowPaidBroadcast   bool
	ReplyParameters      interface{}
	ReplyMarkup          tl.ReplyMarkupClass
	MessageEffectID      *int64
}

// StoryCopy holds options for copying a story with optional modifications to
// caption, privacy, and media areas. Fields that map to optional API flags
// default to their zero values.
//
// Example:
//
//	opt := &params.StoryCopy{
//	    Caption:   "Re-shared story",
//	    ParseMode: params.ParseModeHTML,
//	}
//	err := client.CopyStory(ctx, fromPeer, storyID, opt)
type StoryCopy struct {
	Caption         string
	ParseMode       ParseMode
	CaptionEntities []tl.MessageEntityClass
	Period          *int32
	MediaAreas      interface{}
	Privacy         string
	AllowedUsers    []int64
	DisallowedUsers []int64
	ProtectContent  bool
}

// EditPrivacy holds options for editing the privacy settings of a story.
// Use AllowedUsers and DisallowedUsers to control who can see the story.
//
// Example:
//
//	opt := &params.EditPrivacy{
//	    Privacy:        "contacts",
//	    AllowedUsers:   []int64{12345},
//	}
//	err := client.EditStoryPrivacy(ctx, storyID, opt)
type EditPrivacy struct {
	Privacy         string
	AllowedUsers    []int64
	DisallowedUsers []int64
}

// React holds options for reacting to a message with an emoji. Set Big to
// true to display a large animation.
//
// Example:
//
//	opt := &params.React{
//	    Emoji: "🔥",
//	    Big:   true,
//	}
//	err := msg.React(opt)
type React struct {
	Emoji string
	Big   bool
}

// GiftSend holds options for sending a star gift to a user. Text and
// ParseMode control the attached message.
//
// Example:
//
//	opt := &params.GiftSend{
//	    Text:      "Happy birthday!",
//	    ParseMode: params.ParseModeHTML,
//	    IsPrivate: true,
//	}
//	err := client.SendGift(ctx, userID, giftID, opt)
type GiftSend struct {
	Text          string
	ParseMode     ParseMode
	Entities      []tl.MessageEntityClass
	IsPrivate     bool
	PayForUpgrade bool
}

// BuyGift holds options for purchasing a gift. Set Ton to true to pay with
// TON instead of Telegram Stars.
//
// Example:
//
//	opt := &params.BuyGift{Ton: true}
//	err := client.BuyGift(ctx, giftID, opt)
type BuyGift struct {
	Ton bool
}

// GiftPurchaseOffer holds options for creating a gift purchase offer. Price
// and Duration control the offer terms.
//
// Example:
//
//	opt := &params.GiftPurchaseOffer{
//	    Price:    500,
//	    Duration: 30,
//	}
//	err := client.CreateGiftPurchaseOffer(ctx, opt)
type GiftPurchaseOffer struct {
	Price                int64
	Duration             int32
	PaidMessageStarCount *int64
}
