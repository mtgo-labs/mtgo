package types

import (
	"context"
	"errors"

	"github.com/mtgo-labs/mtgo/telegram/params"
	"github.com/mtgo-labs/mtgo/tg"
)

// errNotImplemented marks bound methods whose server operation is not yet
// available in this build.
var errNotImplemented = errors.New("types: method not implemented")

var (
	// ErrNoBinder is returned when a bound method is called on an object that
	// was not created by a client (e.g. constructed manually or decoded from a
	// cached state).
	ErrNoBinder = errors.New("types: bound methods not available (object was not created by a client)")
	// ErrNoChatBinder is returned when a bound method is called on a Chat that
	// was not created by a client.
	ErrNoChatBinder = errors.New("types: bound methods not available (chat was not created by a client)")
	// ErrNoInlineBinder is returned when a bound method is called on an
	// InlineQuery that was not created by a client.
	ErrNoInlineBinder = errors.New("types: bound methods not available (inline query was not created by a client)")
)

// Each type below exposes bound methods backed by the smallest client
// interface it requires. Interface members carry a context.Context so callers
// can propagate cancellation through the convenience layer; the
// context.Background() forms live on the types themselves.

// SessionBinder is the minimal set of client operations required by ActiveSession bound methods.
// It is satisfied automatically by *telegram.Client.
type SessionBinder interface {
	ResetSession(ctx context.Context, hash int64) error
}

// CallbackQueryBinder is the minimal set of client operations required by CallbackQuery bound methods.
// It is satisfied automatically by *telegram.Client.
type CallbackQueryBinder interface {
	AnswerCallbackQuery(ctx context.Context, callbackQueryID int64, text string, showAlert bool, url string, cacheTime int) error
	DeleteMessages(ctx context.Context, chatID int64, messageIDs []int32, opts ...*params.DeleteMessages) (int, error)
	EditInlineCaption(ctx context.Context, inlineMessageID tg.InputBotInlineMessageIDClass, caption string, opts ...*params.EditInlineOpts) (bool, error)
	EditInlineMedia(ctx context.Context, inlineMessageID tg.InputBotInlineMessageIDClass, media tg.InputMediaClass, opts ...*params.EditInlineOpts) (bool, error)
	EditInlineReplyMarkup(ctx context.Context, inlineMessageID tg.InputBotInlineMessageIDClass, replyMarkup tg.ReplyMarkupClass) (bool, error)
	EditInlineRichMessage(ctx context.Context, inlineMessageID tg.InputBotInlineMessageIDClass, rm tg.InputRichMessageClass, opts ...*params.EditInlineOpts) (bool, error)
	EditInlineText(ctx context.Context, inlineMessageID tg.InputBotInlineMessageIDClass, text string, opts ...*params.EditInlineOpts) (bool, error)
	EditMessageCaption(ctx context.Context, chatID int64, messageID int32, caption string, opts ...*params.EditMessage) (*Message, error)
	EditMessageMedia(ctx context.Context, chatID int64, messageID int32, media tg.InputMediaClass, opts ...*params.EditMessage) (*Message, error)
	EditMessageReplyMarkup(ctx context.Context, chatID int64, messageID int32, replyMarkup tg.ReplyMarkupClass) (*Message, error)
	EditMessageText(ctx context.Context, chatID int64, messageID int32, text string, opts ...*params.EditMessage) (*Message, error)
	SendMessage(ctx context.Context, chatID int64, text string, opts ...*params.SendMessage) (*Message, error)
}

// ChatBinder is the minimal set of client operations required by Chat bound methods.
// It is satisfied automatically by *telegram.Client.
type ChatBinder interface {
	AddChatMember(ctx context.Context, chatID int64, userID int64) error
	ArchiveChat(ctx context.Context, chatID int64) error
	BanChatMember(ctx context.Context, chatID int64, userID int64) error
	DeleteChatPhoto(ctx context.Context, chatID int64) error
	ExportChatInviteLink(ctx context.Context, chatID int64) (string, error)
	GetChat(ctx context.Context, chatID int64) (*Chat, error)
	GetChatEventLog(ctx context.Context, chatID int64, query string, limit int) ([]*ChatEvent, error)
	GetChatMember(ctx context.Context, chatID int64, userID int64) (*ChatMember, error)
	GetChatMembers(ctx context.Context, chatID int64, limit int, offset int) ([]*ChatMember, error)
	JoinChat(ctx context.Context, link string) (*Chat, error)
	LeaveChat(ctx context.Context, chatID int64) error
	MarkChatUnread(ctx context.Context, chatID int64, unread bool) error
	MuteChat(ctx context.Context, chatID int64) error
	PromoteChatMember(ctx context.Context, chatID int64, userID int64, adminRights *tg.ChatAdminRights) error
	RestrictChatMember(ctx context.Context, chatID int64, userID int64, bannedRights *tg.ChatBannedRights) error
	SetAdministratorTitle(ctx context.Context, chatID int64, userID int64, title string) error
	SetChatDescription(ctx context.Context, chatID int64, about string) error
	SetChatPermissions(ctx context.Context, chatID int64, permissions *tg.ChatBannedRights) error
	SetChatPhoto(ctx context.Context, chatID int64, photo tg.InputChatPhotoClass) error
	SetChatTTL(ctx context.Context, chatID int64, ttl int) error
	SetChatTitle(ctx context.Context, chatID int64, title string) error
	SetChatUsername(ctx context.Context, chatID int64, username string) error
	SetProtectedContent(ctx context.Context, chatID int64, enabled bool) error
	SetSlowMode(ctx context.Context, chatID int64, seconds int) error
	UnarchiveChat(ctx context.Context, chatID int64) error
	UnbanChatMember(ctx context.Context, chatID int64, userID int64) error
	UnmuteChat(ctx context.Context, chatID int64) error
	UnpinAllMessages(ctx context.Context, chatID int64) (int, error)
}

// ChatJoinRequestBinder is the minimal set of client operations required by ChatJoinRequest bound methods.
// It is satisfied automatically by *telegram.Client.
type ChatJoinRequestBinder interface {
	ApproveChatJoinRequest(ctx context.Context, chatID int64, userID int64) error
	DeclineChatJoinRequest(ctx context.Context, chatID int64, userID int64) error
}

// GiftBinder is the minimal set of client operations required by Gift bound methods.
// It is satisfied automatically by *telegram.Client.
type GiftBinder interface {
	ConvertGift(ctx context.Context, gift tg.InputSavedStarGiftClass) error
	HideGift(ctx context.Context, gift tg.InputSavedStarGiftClass) error
	ShowGift(ctx context.Context, gift tg.InputSavedStarGiftClass) error
	TransferGift(ctx context.Context, gift tg.InputSavedStarGiftClass, recipientID int64) (*Message, error)
	UpgradeGift(ctx context.Context, gift tg.InputSavedStarGiftClass, keepOriginalDetails bool) (*Message, error)
}

// InlineQueryBinder is the minimal set of client operations required by InlineQuery bound methods.
// It is satisfied automatically by *telegram.Client.
type InlineQueryBinder interface {
	AnswerInlineQuery(ctx context.Context, queryID int64, results []tg.InputBotInlineResultClass, opts ...*params.AnswerInlineQueryOption) error
}

// MessageBinder is the minimal set of client operations required by Message bound methods.
// It is satisfied automatically by *telegram.Client.
type MessageBinder interface {
	CopyMediaGroup(ctx context.Context, chatID int64, fromChatID int64, groupedID int64) ([]*Message, error)
	CopyMessage(ctx context.Context, chatID int64, fromChatID int64, messageID int32, opts ...*params.CopyMessage) (int64, error)
	DeleteMessages(ctx context.Context, chatID int64, messageIDs []int32, opts ...*params.DeleteMessages) (int, error)
	DownloadMedia(ctx context.Context, media Media, thumbSize string, opts *params.Download) ([]byte, error)
	EditMessageCaption(ctx context.Context, chatID int64, messageID int32, caption string, opts ...*params.EditMessage) (*Message, error)
	EditMessageMedia(ctx context.Context, chatID int64, messageID int32, media tg.InputMediaClass, opts ...*params.EditMessage) (*Message, error)
	EditMessageReplyMarkup(ctx context.Context, chatID int64, messageID int32, replyMarkup tg.ReplyMarkupClass) (*Message, error)
	EditMessageText(ctx context.Context, chatID int64, messageID int32, text string, opts ...*params.EditMessage) (*Message, error)
	ForwardMessages(ctx context.Context, chatID int64, fromChatID int64, messageIDs []int32, opts ...*params.ForwardMessages) ([]*Message, error)
	GetMediaGroup(ctx context.Context, chatID int64, messageID int32) ([]*Message, error)
	GetMessages(ctx context.Context, chatID int64, messageIDs []int32) ([]*Message, error)
	PinMessage(ctx context.Context, chatID int64, messageID int32, opts ...*params.PinMessage) (*Message, error)
	RandomID() int64
	ReadChatHistory(ctx context.Context, chatID int64, maxID int32) error
	RetractVote(ctx context.Context, chatID int64, messageID int32) error
	SendAnimation(ctx context.Context, chatID int64, file *InputFile, caption string, opts ...*params.SendAnimation) (*Message, error)
	SendAudio(ctx context.Context, chatID int64, file *InputFile, caption string, opts ...*params.SendAudio) (*Message, error)
	SendChatAction(ctx context.Context, chatID int64, action tg.SendMessageActionClass) error
	SendContact(ctx context.Context, chatID int64, phoneNumber, firstName, lastName string, opts *params.SendMessage) (*Message, error)
	SendDocument(ctx context.Context, chatID int64, file *InputFile, caption string, opts ...*params.SendDocument) (*Message, error)
	SendGame(ctx context.Context, chatID int64, gameShortName string, opts ...*params.SendMessage) (*Message, error)
	SendInlineBotResult(ctx context.Context, chatID int64, queryID int64, resultID string, opts ...*params.SendInlineBotResultOption) (*Message, error)
	SendLocation(ctx context.Context, chatID int64, lat, lng float64, opts *params.SendMessage) (*Message, error)
	SendMedia(ctx context.Context, chatID int64, media tg.InputMediaClass, caption string, opts ...*params.SendMessage) (*Message, error)
	SendMediaGroup(ctx context.Context, chatID int64, items []*tg.InputSingleMedia, opts ...*params.SendMessage) ([]*Message, error)
	SendMessage(ctx context.Context, chatID int64, text string, opts ...*params.SendMessage) (*Message, error)
	SendPhoto(ctx context.Context, chatID int64, file *InputFile, caption string, opts ...*params.SendPhoto) (*Message, error)
	SendPoll(ctx context.Context, chatID int64, question string, options []string, opts *params.SendMessage) (*Message, error)
	SendReaction(ctx context.Context, chatID int64, messageID int32, reactions []Reaction, opts ...*params.SendReactionOption) error
	SendRichMessage(ctx context.Context, chatID int64, rm tg.InputRichMessageClass, opts ...*params.SendMessage) (*Message, error)
	SendSticker(ctx context.Context, chatID int64, file *InputFile, opts ...*params.SendSticker) (*Message, error)
	SendVenue(ctx context.Context, chatID int64, lat, lng float64, title, address string, opts *params.SendMessage) (*Message, error)
	SendVideo(ctx context.Context, chatID int64, file *InputFile, caption string, opts ...*params.SendVideo) (*Message, error)
	SendVideoNote(ctx context.Context, chatID int64, file *InputFile, opts ...*params.SendVideoNote) (*Message, error)
	SendVoice(ctx context.Context, chatID int64, file *InputFile, caption string, opts ...*params.SendVoice) (*Message, error)
	SummarizeMessage(ctx context.Context, chatID int64, messageID int32) (*tg.TextWithEntities, error)
	UnpinMessage(ctx context.Context, chatID int64, messageID int32) (*Message, error)
	VotePoll(ctx context.Context, chatID int64, messageID int32, options [][]byte) error
}

// PreCheckoutQueryBinder is the minimal set of client operations required by PreCheckoutQuery bound methods.
// It is satisfied automatically by *telegram.Client.
type PreCheckoutQueryBinder interface {
	AnswerPreCheckoutQuery(ctx context.Context, queryID int64, ok bool, errorMessage string) error
}

// ShippingQueryBinder is the minimal set of client operations required by ShippingQuery bound methods.
// It is satisfied automatically by *telegram.Client.
type ShippingQueryBinder interface {
	AnswerShippingQuery(ctx context.Context, queryID int64, ok bool, options []*tg.ShippingOption) error
}

// StoryBinder is the minimal set of client operations required by Story bound methods.
// It is satisfied automatically by *telegram.Client.
type StoryBinder interface {
	DeleteStories(ctx context.Context, chatID int64, storyIDs []int32) error
	DownloadMedia(ctx context.Context, media Media, thumbSize string, opts *params.Download) ([]byte, error)
	EditStoryCaption(ctx context.Context, chatID int64, storyID int32, caption string) (*Story, error)
	EditStoryMedia(ctx context.Context, chatID int64, storyID int32, media tg.InputMediaClass) (*Story, error)
	EditStoryPrivacy(ctx context.Context, peerID int64, storyID int32, opts ...*params.EditPrivacy) (*Story, error)
	ForwardStory(ctx context.Context, targetChatID int64, sourceChatID int64, storyID int32) (*Message, error)
	GetStories(ctx context.Context, userID int64, storyIDs []int32) ([]*Story, error)
	ReadStories(ctx context.Context, peerID int64, maxStoryID int32) error
	SendMedia(ctx context.Context, chatID int64, media tg.InputMediaClass, caption string, opts ...*params.SendMessage) (*Message, error)
	SendMessage(ctx context.Context, chatID int64, text string, opts ...*params.SendMessage) (*Message, error)
	SendStoryReaction(ctx context.Context, peerID int64, storyID int32, opts ...*params.React) error
}

// UserBinder is the minimal set of client operations required by User bound methods.
// It is satisfied automatically by *telegram.Client.
type UserBinder interface {
	ArchiveChat(ctx context.Context, chatID int64) error
	BlockUser(ctx context.Context, userID int64) error
	GetCommonChats(ctx context.Context, userID int64, limit int) ([]*Chat, error)
	UnarchiveChat(ctx context.Context, chatID int64) error
	UnblockUser(ctx context.Context, userID int64) error
}

// FolderBinder is the minimal set of client operations required by Folder
// bound methods. It is satisfied automatically by *telegram.Client.
type FolderBinder interface {
	BoundDeleteFolder(ctx context.Context, folderID int32) error
	BoundEditFolder(ctx context.Context, filter *tg.DialogFilter) error
	BoundExcludeChat(ctx context.Context, folderID int32, chatID int64) error
	BoundIncludeChat(ctx context.Context, folderID int32, chatID int64) error
	BoundPinChatInFolder(ctx context.Context, folderID int32, chatID int64) error
	BoundRemoveChatFromFolder(ctx context.Context, folderID int32, chatID int64) error
	BoundUpdateFolderColor(ctx context.Context, folderID int32, color int32) error
}

// editInlineOptsFromParams converts edit-message option structs into the
// inline-edit option form used by the inline editing operations.
func editInlineOptsFromParams(opts ...*params.EditMessage) []*params.EditInlineOpts {
	if len(opts) == 0 || opts[0] == nil {
		return nil
	}
	p := opts[0]
	return []*params.EditInlineOpts{{
		NoWebpage:   p.DisableWebPagePreview,
		InvertMedia: p.InvertMedia,
		ReplyMarkup: p.ReplyMarkup,
		Entities:    p.Entities,
		ParseMode:   p.ParseMode,
		RichMessage: p.RichMessage,
	}}
}
