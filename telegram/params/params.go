// Package params defines option and parameter structs used to configure
// Telegram Bot API calls such as sending, editing, forwarding, copying,
// deleting, and pinning messages, as well as file upload/download progress.
package params

import (
	"fmt"
	"reflect"

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
	v := reflect.ValueOf(opt)
	if v.Kind() == reflect.Pointer {
		return !v.IsNil()
	}
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
	// DisableWebPagePreview suppresses link previews in the sent message.
	DisableWebPagePreview bool

	// DisableNotification silently delivers the message without triggering
	// a push notification on the recipient's device.
	DisableNotification bool

	// Silent is an alias for DisableNotification kept for backwards
	// compatibility with earlier API versions.
	Silent bool

	// Background sends the message as a background action, avoiding
	// generation of a notification on the sender's side.
	Background bool

	// ClearDraft clears the current text draft in the chat after the
	// message is sent successfully.
	ClearDraft bool

	// NoForwards disables forwarding of the sent message by recipients
	// (also known as "forwards banned").
	NoForwards bool

	// InvertMedia moves the media preview above the caption text instead
	// of below it.
	InvertMedia bool

	// ReplyToMessageID is the ID of a message to reply to. A value of 0
	// means no reply.
	ReplyToMessageID int32

	// ReplyTo is an explicit reply target (message, story, mono-forum).
	// When set, it overrides ReplyToMessageID.
	ReplyTo tl.InputReplyToClass

	// ReplyMarkup provides an inline or reply keyboard to attach to the
	// message. Nil means no markup.
	ReplyMarkup tl.ReplyMarkupClass

	// Entities supplies pre-formatted message entities. When non-empty,
	// the API uses these entities directly instead of parsing the text.
	Entities []tl.MessageEntityClass

	// ParseMode determines how the message text is parsed (Markdown, HTML,
	// etc.). See the ParseMode constants for available values.
	ParseMode ParseMode

	// ScheduleDate is an optional Unix timestamp at which the message
	// should be delivered. Nil sends immediately.
	ScheduleDate *int32

	// EffectID is an optional message-effect identifier applied to the
	// message for visual effects in supported clients.
	EffectID *int64

	// SendAs specifies the peer (channel, group) from which the message
	// should appear to be sent. Nil means send as the current user.
	SendAs tl.InputPeerClass
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
	// DisableWebPagePreview suppresses link previews in the edited message.
	DisableWebPagePreview bool

	// InvertMedia moves the media preview above the caption text instead
	// of below it.
	InvertMedia bool

	// ReplyMarkup updates the inline or reply keyboard attached to the
	// message. Nil leaves the existing markup unchanged.
	ReplyMarkup tl.ReplyMarkupClass

	// ParseMode determines how the updated message text is parsed.
	ParseMode ParseMode

	// Entities supplies pre-formatted message entities for the updated
	// text. When non-empty, the API uses these entities directly.
	Entities []tl.MessageEntityClass

	// ScheduleDate is an optional Unix timestamp to reschedule the
	// message. Nil keeps the current schedule.
	ScheduleDate *int32
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
	// DisableNotification silently delivers the forwarded messages without
	// triggering push notifications on the recipients' devices.
	DisableNotification bool

	// NoForwards disables further forwarding of the forwarded messages by
	// recipients.
	NoForwards bool

	// DropAuthor removes the original author attribution from the
	// forwarded messages, making them appear as if sent by the forwarder.
	DropAuthor bool

	// DropMediaCaptions strips captions from media in the forwarded
	// messages.
	DropMediaCaptions bool

	// ScheduleDate is an optional Unix timestamp at which the forwarded
	// messages should be delivered. Nil forwards immediately.
	ScheduleDate *int32
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
	// Caption overrides the media caption on the copied message. An empty
	// string preserves the original caption.
	Caption string

	// DisableNotification silently delivers the copied message without
	// triggering a push notification.
	DisableNotification bool

	// ReplyToMessageID is the ID of a message to reply to. A value of 0
	// means no reply.
	ReplyToMessageID int32

	// ReplyMarkup provides an inline or reply keyboard to attach to the
	// copied message. Nil means no markup.
	ReplyMarkup tl.ReplyMarkupClass

	// ScheduleDate is an optional Unix timestamp at which the copied
	// message should be delivered. Nil sends immediately.
	ScheduleDate *int32

	// DropAuthor removes the original author attribution when copying.
	DropAuthor bool
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
	ExcludeSaved         bool
	ExcludeUnlimited     bool
	ExcludeUnique        bool
	SortByValue          bool
	ExcludeUpgradable    bool
	ExcludeUnupgradable  bool
	PeerColorAvailable   bool
	ExcludeHosted        bool
	CollectionID         int32
	Offset               string
	Limit                int32
}

type SendPoll struct {
	DisableNotification   bool
	Silent                bool
	Background            bool
	ClearDraft            bool
	NoForwards            bool
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
	ClosePeriod           *int32
	CloseDate             *int32
	CorrectAnswers        [][]byte
	Solution              *string
	SolutionEntities      []tl.MessageEntityClass
}

type SendVenue struct {
	DisableNotification bool
	Silent              bool
	Background          bool
	ClearDraft          bool
	NoForwards          bool
	ReplyToMessageID    int32
	ReplyTo             tl.InputReplyToClass
	ReplyMarkup         tl.ReplyMarkupClass
	ScheduleDate        *int32
	EffectID            *int64
	SendAs              tl.InputPeerClass
	Provider            string
	VenueID             string
	VenueType           string
}

type SendContact struct {
	DisableNotification bool
	Silent              bool
	Background          bool
	ClearDraft          bool
	NoForwards          bool
	ReplyToMessageID    int32
	ReplyTo             tl.InputReplyToClass
	ReplyMarkup         tl.ReplyMarkupClass
	ScheduleDate        *int32
	EffectID            *int64
	SendAs              tl.InputPeerClass
	Vcard               string
}

type SendLocation struct {
	DisableNotification bool
	Silent              bool
	Background          bool
	ClearDraft          bool
	NoForwards          bool
	ReplyToMessageID    int32
	ReplyTo             tl.InputReplyToClass
	ReplyMarkup         tl.ReplyMarkupClass
	ScheduleDate        *int32
	EffectID            *int64
	SendAs              tl.InputPeerClass
	AccuracyRadius      *int32
}

type SendDice struct {
	DisableNotification bool
	Silent              bool
	Background          bool
	ClearDraft          bool
	NoForwards          bool
	ReplyToMessageID    int32
	ReplyTo             tl.InputReplyToClass
	ReplyMarkup         tl.ReplyMarkupClass
	ScheduleDate        *int32
	EffectID            *int64
	SendAs              tl.InputPeerClass
	Emoticon            string
}

type SendGame struct {
	DisableNotification bool
	Silent              bool
	Background          bool
	ClearDraft          bool
	NoForwards          bool
	ReplyToMessageID    int32
	ReplyTo             tl.InputReplyToClass
	ReplyMarkup         tl.ReplyMarkupClass
	ScheduleDate        *int32
	EffectID            *int64
	SendAs              tl.InputPeerClass
}

type SendMediaGroup struct {
	DisableNotification bool
	Silent              bool
	Background          bool
	ClearDraft          bool
	NoForwards          bool
	ReplyToMessageID    int32
	ReplyTo             tl.InputReplyToClass
	ScheduleDate        *int32
	EffectID            *int64
	SendAs              tl.InputPeerClass
}

type SendChecklist struct {
	DisableNotification bool
	Silent              bool
	Background          bool
	ClearDraft          bool
	NoForwards          bool
	ReplyToMessageID    int32
	ReplyTo             tl.InputReplyToClass
	ReplyMarkup         tl.ReplyMarkupClass
	ScheduleDate        *int32
	EffectID            *int64
	SendAs              tl.InputPeerClass
	OthersCanAppend     bool
	OthersCanComplete   bool
	RepeatPeriod        *int32
	PaidMessageStars    *int64
}

func (s *SendChecklist) getFlatSendFields() (bool, bool, bool, bool, bool, int32, tl.InputReplyToClass, tl.ReplyMarkupClass, *int32, *int64, tl.InputPeerClass) {
	return s.DisableNotification, s.Silent, s.Background, s.ClearDraft, s.NoForwards, s.ReplyToMessageID, s.ReplyTo, s.ReplyMarkup, s.ScheduleDate, s.EffectID, s.SendAs
}
func (s *SendChecklist) ToSendMsg() *SendMessage { return flatToSendMsg(s) }

type SendInlineBotResult struct {
	DisableNotification bool
	Silent              bool
	Background          bool
	ClearDraft          bool
	NoForwards          bool
	ReplyToMessageID    int32
	ReplyTo             tl.InputReplyToClass
	ReplyMarkup         tl.ReplyMarkupClass
	ScheduleDate        *int32
	EffectID            *int64
	SendAs              tl.InputPeerClass
	HideVia             bool
	AllowPaidStars      *int64
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

type SendAudio struct {
	DisableNotification bool
	Silent              bool
	Background          bool
	ClearDraft          bool
	NoForwards          bool
	ReplyToMessageID    int32
	ReplyTo             tl.InputReplyToClass
	ReplyMarkup         tl.ReplyMarkupClass
	ScheduleDate        *int32
	EffectID            *int64
	SendAs              tl.InputPeerClass
	Duration            int32
	Performer           string
	Title               string
	FileName            string
	Thumb               string
}

func (s *SendAudio) getFlatSendFields() (bool, bool, bool, bool, bool, int32, tl.InputReplyToClass, tl.ReplyMarkupClass, *int32, *int64, tl.InputPeerClass) {
	return s.DisableNotification, s.Silent, s.Background, s.ClearDraft, s.NoForwards, s.ReplyToMessageID, s.ReplyTo, s.ReplyMarkup, s.ScheduleDate, s.EffectID, s.SendAs
}
func (s *SendAudio) ToSendMsg() *SendMessage { return flatToSendMsg(s) }

type SendVideo struct {
	DisableNotification bool
	Silent              bool
	Background          bool
	ClearDraft          bool
	NoForwards          bool
	ReplyToMessageID    int32
	ReplyTo             tl.InputReplyToClass
	ReplyMarkup         tl.ReplyMarkupClass
	ScheduleDate        *int32
	EffectID            *int64
	SendAs              tl.InputPeerClass
	Duration            float64
	Width               int32
	Height              int32
	SupportsStreaming   bool
	FileName            string
	Thumb               string
}

func (s *SendVideo) getFlatSendFields() (bool, bool, bool, bool, bool, int32, tl.InputReplyToClass, tl.ReplyMarkupClass, *int32, *int64, tl.InputPeerClass) {
	return s.DisableNotification, s.Silent, s.Background, s.ClearDraft, s.NoForwards, s.ReplyToMessageID, s.ReplyTo, s.ReplyMarkup, s.ScheduleDate, s.EffectID, s.SendAs
}
func (s *SendVideo) ToSendMsg() *SendMessage { return flatToSendMsg(s) }

type SendDocument struct {
	DisableNotification bool
	Silent              bool
	Background          bool
	ClearDraft          bool
	NoForwards          bool
	ReplyToMessageID    int32
	ReplyTo             tl.InputReplyToClass
	ReplyMarkup         tl.ReplyMarkupClass
	ScheduleDate        *int32
	EffectID            *int64
	SendAs              tl.InputPeerClass
	FileName            string
	Thumb               string
	MimeType            string
}

func (s *SendDocument) getFlatSendFields() (bool, bool, bool, bool, bool, int32, tl.InputReplyToClass, tl.ReplyMarkupClass, *int32, *int64, tl.InputPeerClass) {
	return s.DisableNotification, s.Silent, s.Background, s.ClearDraft, s.NoForwards, s.ReplyToMessageID, s.ReplyTo, s.ReplyMarkup, s.ScheduleDate, s.EffectID, s.SendAs
}
func (s *SendDocument) ToSendMsg() *SendMessage { return flatToSendMsg(s) }

type SendPhoto struct {
	DisableNotification bool
	Silent              bool
	Background          bool
	ClearDraft          bool
	NoForwards          bool
	ReplyToMessageID    int32
	ReplyTo             tl.InputReplyToClass
	ReplyMarkup         tl.ReplyMarkupClass
	ScheduleDate        *int32
	EffectID            *int64
	SendAs              tl.InputPeerClass
	FileName            string
}

func (s *SendPhoto) getFlatSendFields() (bool, bool, bool, bool, bool, int32, tl.InputReplyToClass, tl.ReplyMarkupClass, *int32, *int64, tl.InputPeerClass) {
	return s.DisableNotification, s.Silent, s.Background, s.ClearDraft, s.NoForwards, s.ReplyToMessageID, s.ReplyTo, s.ReplyMarkup, s.ScheduleDate, s.EffectID, s.SendAs
}
func (s *SendPhoto) ToSendMsg() *SendMessage { return flatToSendMsg(s) }

type SendAnimation struct {
	DisableNotification bool
	Silent              bool
	Background          bool
	ClearDraft          bool
	NoForwards          bool
	ReplyToMessageID    int32
	ReplyTo             tl.InputReplyToClass
	ReplyMarkup         tl.ReplyMarkupClass
	ScheduleDate        *int32
	EffectID            *int64
	SendAs              tl.InputPeerClass
	FileName            string
	Thumb               string
}

func (s *SendAnimation) getFlatSendFields() (bool, bool, bool, bool, bool, int32, tl.InputReplyToClass, tl.ReplyMarkupClass, *int32, *int64, tl.InputPeerClass) {
	return s.DisableNotification, s.Silent, s.Background, s.ClearDraft, s.NoForwards, s.ReplyToMessageID, s.ReplyTo, s.ReplyMarkup, s.ScheduleDate, s.EffectID, s.SendAs
}
func (s *SendAnimation) ToSendMsg() *SendMessage { return flatToSendMsg(s) }

type SendVoice struct {
	DisableNotification bool
	Silent              bool
	Background          bool
	ClearDraft          bool
	NoForwards          bool
	ReplyToMessageID    int32
	ReplyTo             tl.InputReplyToClass
	ReplyMarkup         tl.ReplyMarkupClass
	ScheduleDate        *int32
	EffectID            *int64
	SendAs              tl.InputPeerClass
	Duration            int32
	FileName            string
}

func (s *SendVoice) getFlatSendFields() (bool, bool, bool, bool, bool, int32, tl.InputReplyToClass, tl.ReplyMarkupClass, *int32, *int64, tl.InputPeerClass) {
	return s.DisableNotification, s.Silent, s.Background, s.ClearDraft, s.NoForwards, s.ReplyToMessageID, s.ReplyTo, s.ReplyMarkup, s.ScheduleDate, s.EffectID, s.SendAs
}
func (s *SendVoice) ToSendMsg() *SendMessage { return flatToSendMsg(s) }

type SendVideoNote struct {
	DisableNotification bool
	Silent              bool
	Background          bool
	ClearDraft          bool
	NoForwards          bool
	ReplyToMessageID    int32
	ReplyTo             tl.InputReplyToClass
	ReplyMarkup         tl.ReplyMarkupClass
	ScheduleDate        *int32
	EffectID            *int64
	SendAs              tl.InputPeerClass
	Duration            float64
	FileName            string
	Thumb               string
}

func (s *SendVideoNote) getFlatSendFields() (bool, bool, bool, bool, bool, int32, tl.InputReplyToClass, tl.ReplyMarkupClass, *int32, *int64, tl.InputPeerClass) {
	return s.DisableNotification, s.Silent, s.Background, s.ClearDraft, s.NoForwards, s.ReplyToMessageID, s.ReplyTo, s.ReplyMarkup, s.ScheduleDate, s.EffectID, s.SendAs
}
func (s *SendVideoNote) ToSendMsg() *SendMessage { return flatToSendMsg(s) }

type SendSticker struct {
	DisableNotification bool
	Silent              bool
	Background          bool
	ClearDraft          bool
	NoForwards          bool
	ReplyToMessageID    int32
	ReplyTo             tl.InputReplyToClass
	ReplyMarkup         tl.ReplyMarkupClass
	ScheduleDate        *int32
	EffectID            *int64
	SendAs              tl.InputPeerClass
	FileName            string
}

func (s *SendSticker) getFlatSendFields() (bool, bool, bool, bool, bool, int32, tl.InputReplyToClass, tl.ReplyMarkupClass, *int32, *int64, tl.InputPeerClass) {
	return s.DisableNotification, s.Silent, s.Background, s.ClearDraft, s.NoForwards, s.ReplyToMessageID, s.ReplyTo, s.ReplyMarkup, s.ScheduleDate, s.EffectID, s.SendAs
}
func (s *SendSticker) ToSendMsg() *SendMessage { return flatToSendMsg(s) }
