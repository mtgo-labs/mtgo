package types

import (
	"context"
	"errors"

	"github.com/mtgo-labs/mtgo/telegram/params"
	"github.com/mtgo-labs/mtgo/tg"
)

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
// interface it requires. The interface members carry a context.Context so
// callers can propagate cancellation through the convenience layer; the
// context.Background() forms live on the types themselves.

// SessionBinder is the minimal set of client operations required by ActiveSession bound methods. It is satisfied automatically by *telegram.Client.
type SessionBinder interface {
	BoundResetSession(ctx context.Context, hash int64) error
}

// CallbackQueryBinder is the minimal set of client operations required by CallbackQuery bound methods. It is satisfied automatically by *telegram.Client.
type CallbackQueryBinder interface {
	BoundAnswerCallback(ctx context.Context, queryID int64, opts ...*params.AnswerCallback) error
	BoundDelete(ctx context.Context, chatID int64, msgIDs []int32, opts ...*params.DeleteMessages) (int, error)
	BoundEdit(ctx context.Context, chatID int64, msgID int32, text string, opts ...*params.EditMessage) (*Message, error)
	BoundEditCaption(ctx context.Context, chatID int64, msgID int32, caption string, opts ...*params.EditMessage) (*Message, error)
	BoundEditInline(ctx context.Context, inlineMessageID tg.InputBotInlineMessageIDClass, text string, opts ...*params.EditMessage) (bool, error)
	BoundEditInlineCaption(ctx context.Context, inlineMessageID tg.InputBotInlineMessageIDClass, caption string, opts ...*params.EditMessage) (bool, error)
	BoundEditInlineMedia(ctx context.Context, inlineMessageID tg.InputBotInlineMessageIDClass, media tg.InputMediaClass) (bool, error)
	BoundEditInlineReplyMarkup(ctx context.Context, inlineMessageID tg.InputBotInlineMessageIDClass, markup tg.ReplyMarkupClass) (bool, error)
	BoundEditInlineRich(ctx context.Context, inlineMessageID tg.InputBotInlineMessageIDClass, rm tg.InputRichMessageClass) (bool, error)
	BoundEditMedia(ctx context.Context, chatID int64, msgID int32, media tg.InputMediaClass) (*Message, error)
	BoundEditReplyMarkup(ctx context.Context, chatID int64, msgID int32, markup tg.ReplyMarkupClass) (*Message, error)
	BoundEditRich(ctx context.Context, chatID int64, msgID int32, rm tg.InputRichMessageClass) (*Message, error)
	BoundSend(ctx context.Context, chatID int64, text string, replyTo int32, opts ...*params.SendMessage) (*Message, error)
}

// ChatBinder is the minimal set of client operations required by Chat bound methods. It is satisfied automatically by *telegram.Client.
type ChatBinder interface {
	BoundAddMembers(ctx context.Context, chatID int64, userID int64) error
	BoundArchive(ctx context.Context, chatID int64) error
	BoundBanMember(ctx context.Context, chatID int64, userID int64) error
	BoundDeletePhoto(ctx context.Context, chatID int64) error
	BoundExportInviteLink(ctx context.Context, chatID int64) (string, error)
	BoundGetChat(ctx context.Context, chatID int64) (*Chat, error)
	BoundGetEventLog(ctx context.Context, chatID int64, query string, limit int) ([]*ChatEvent, error)
	BoundGetMember(ctx context.Context, chatID int64, userID int64) (*ChatMember, error)
	BoundGetMembers(ctx context.Context, chatID int64, limit int, offset int) ([]*ChatMember, error)
	BoundJoinChat(ctx context.Context, chatID int64, link string) (*Chat, error)
	BoundLeaveChat(ctx context.Context, chatID int64) error
	BoundMarkUnread(ctx context.Context, chatID int64, unread bool) error
	BoundMute(ctx context.Context, chatID int64) error
	BoundPromoteMember(ctx context.Context, chatID int64, userID int64, adminRights *tg.ChatAdminRights) error
	BoundRestrictMember(ctx context.Context, chatID int64, userID int64, bannedRights *tg.ChatBannedRights) error
	BoundSetAdminTitle(ctx context.Context, chatID int64, userID int64, title string) error
	BoundSetDescription(ctx context.Context, chatID int64, description string) error
	BoundSetPermissions(ctx context.Context, chatID int64, permissions *tg.ChatBannedRights) error
	BoundSetPhoto(ctx context.Context, chatID int64, photo tg.InputChatPhotoClass) error
	BoundSetProtectedContent(ctx context.Context, chatID int64, enabled bool) error
	BoundSetSlowMode(ctx context.Context, chatID int64, seconds int) error
	BoundSetTTL(ctx context.Context, chatID int64, ttl int) error
	BoundSetTitle(ctx context.Context, chatID int64, title string) error
	BoundSetUsername(ctx context.Context, chatID int64, username string) error
	BoundUnarchive(ctx context.Context, chatID int64) error
	BoundUnbanMember(ctx context.Context, chatID int64, userID int64) error
	BoundUnmute(ctx context.Context, chatID int64) error
	BoundUnpinAll(ctx context.Context, chatID int64) (int, error)
}

// ChatJoinRequestBinder is the minimal set of client operations required by ChatJoinRequest bound methods. It is satisfied automatically by *telegram.Client.
type ChatJoinRequestBinder interface {
	BoundApproveJoinRequest(ctx context.Context, chatID int64, userID int64) error
	BoundDeclineJoinRequest(ctx context.Context, chatID int64, userID int64) error
}

// FolderBinder is the minimal set of client operations required by Folder bound methods. It is satisfied automatically by *telegram.Client.
type FolderBinder interface {
	BoundDeleteFolder(ctx context.Context, folderID int32) error
	BoundEditFolder(ctx context.Context, filter *tg.DialogFilter) error
	BoundExcludeChat(ctx context.Context, folderID int32, chatID int64) error
	BoundIncludeChat(ctx context.Context, folderID int32, chatID int64) error
	BoundPinChatInFolder(ctx context.Context, folderID int32, chatID int64) error
	BoundRemoveChatFromFolder(ctx context.Context, folderID int32, chatID int64) error
	BoundUpdateFolderColor(ctx context.Context, folderID int32, color int32) error
}

// GiftBinder is the minimal set of client operations required by Gift bound methods. It is satisfied automatically by *telegram.Client.
type GiftBinder interface {
	BoundConvertGift(ctx context.Context, msgID int32) error
	BoundHideGift(ctx context.Context, msgID int32) error
	BoundShowGift(ctx context.Context, msgID int32) error
	BoundTransferGift(ctx context.Context, msgID int32, toID int64) error
	BoundUpgradeGift(ctx context.Context, msgID int32, keepOriginalDetails bool) error
}

// InlineQueryBinder is the minimal set of client operations required by InlineQuery bound methods. It is satisfied automatically by *telegram.Client.
type InlineQueryBinder interface {
	BoundAnswerInline(ctx context.Context, queryID int64, results []tg.InputBotInlineResultClass, opts ...*params.InlineQuery) error
}

// MessageBinder is the minimal set of client operations required by Message bound methods. It is satisfied automatically by *telegram.Client.
type MessageBinder interface {
	BoundAnswerAnimation(ctx context.Context, chatID int64, file *InputFile, caption string, opts ...*params.SendAnimation) (*Message, error)
	BoundAnswerAudio(ctx context.Context, chatID int64, file *InputFile, caption string, opts ...*params.SendAudio) (*Message, error)
	BoundAnswerDocument(ctx context.Context, chatID int64, file *InputFile, caption string, opts ...*params.SendDocument) (*Message, error)
	BoundAnswerPhoto(ctx context.Context, chatID int64, file *InputFile, caption string, opts ...*params.SendPhoto) (*Message, error)
	BoundAnswerSticker(ctx context.Context, chatID int64, file *InputFile, opts ...*params.SendSticker) (*Message, error)
	BoundAnswerVideo(ctx context.Context, chatID int64, file *InputFile, caption string, opts ...*params.SendVideo) (*Message, error)
	BoundAnswerVideoNote(ctx context.Context, chatID int64, file *InputFile, opts ...*params.SendVideoNote) (*Message, error)
	BoundAnswerVoice(ctx context.Context, chatID int64, file *InputFile, caption string, opts ...*params.SendVoice) (*Message, error)
	BoundCopy(ctx context.Context, chatID int64, fromChatID int64, msgID int32, opts ...*params.CopyMessage) (int64, error)
	BoundCopyMediaGroup(ctx context.Context, chatID int64, fromChatID int64, msgID int32) ([]*Message, error)
	BoundDelete(ctx context.Context, chatID int64, msgIDs []int32, opts ...*params.DeleteMessages) (int, error)
	BoundDownload(ctx context.Context, chatID int64, msgID int32, opts ...*params.Download) ([]byte, error)
	BoundDownloadTo(ctx context.Context, chatID int64, msgID int32, fileName string, opts ...*params.Download) (string, error)
	BoundEdit(ctx context.Context, chatID int64, msgID int32, text string, opts ...*params.EditMessage) (*Message, error)
	BoundEditCaption(ctx context.Context, chatID int64, msgID int32, caption string, opts ...*params.EditMessage) (*Message, error)
	BoundEditMedia(ctx context.Context, chatID int64, msgID int32, media tg.InputMediaClass) (*Message, error)
	BoundEditReplyMarkup(ctx context.Context, chatID int64, msgID int32, markup tg.ReplyMarkupClass) (*Message, error)
	BoundForward(ctx context.Context, chatID int64, fromChatID int64, msgID int32, opts ...*params.ForwardMessages) (*Message, error)
	BoundGetMediaGroup(ctx context.Context, chatID int64, msgID int32) ([]*Message, error)
	BoundPin(ctx context.Context, chatID int64, msgID int32, opts ...*params.PinMessage) error
	BoundReact(ctx context.Context, chatID int64, msgID int32, opts ...*params.React) error
	BoundRead(ctx context.Context, chatID int64, msgID int32) error
	BoundReplyAnimation(ctx context.Context, chatID int64, file *InputFile, caption string, replyTo int32, opts ...*params.SendAnimation) (*Message, error)
	BoundReplyAudio(ctx context.Context, chatID int64, file *InputFile, caption string, replyTo int32, opts ...*params.SendAudio) (*Message, error)
	BoundReplyChecklist(ctx context.Context, chatID int64, checklist *tg.InputMediaTodo, replyTo int32, opts ...*params.SendChecklist) (*Message, error)
	BoundReplyDocument(ctx context.Context, chatID int64, file *InputFile, caption string, replyTo int32, opts ...*params.SendDocument) (*Message, error)
	BoundReplyPhoto(ctx context.Context, chatID int64, file *InputFile, caption string, replyTo int32, opts ...*params.SendPhoto) (*Message, error)
	BoundReplySticker(ctx context.Context, chatID int64, file *InputFile, replyTo int32, opts ...*params.SendSticker) (*Message, error)
	BoundReplyVideo(ctx context.Context, chatID int64, file *InputFile, caption string, replyTo int32, opts ...*params.SendVideo) (*Message, error)
	BoundReplyVideoNote(ctx context.Context, chatID int64, file *InputFile, replyTo int32, opts ...*params.SendVideoNote) (*Message, error)
	BoundReplyVoice(ctx context.Context, chatID int64, file *InputFile, caption string, replyTo int32, opts ...*params.SendVoice) (*Message, error)
	BoundRetractVote(ctx context.Context, chatID int64, msgID int32) error
	BoundSend(ctx context.Context, chatID int64, text string, replyTo int32, opts ...*params.SendMessage) (*Message, error)
	BoundSendChatAction(ctx context.Context, chatID int64, action tg.SendMessageActionClass) error
	BoundSendContact(ctx context.Context, chatID int64, phone, firstName, lastName string, replyTo int32, opts ...*params.SendContact) (*Message, error)
	BoundSendDice(ctx context.Context, chatID int64, emoji string, replyTo int32, opts ...*params.SendDice) (*Message, error)
	BoundSendGame(ctx context.Context, chatID int64, gameShortName string, replyTo int32, opts ...*params.SendGame) (*Message, error)
	BoundSendInlineBotResult(ctx context.Context, chatID int64, queryID int64, resultID string, replyTo int32, opts ...*params.SendInlineBotResult) (*Message, error)
	BoundSendLocation(ctx context.Context, chatID int64, lat, lng float64, replyTo int32, opts ...*params.SendLocation) (*Message, error)
	BoundSendMedia(ctx context.Context, chatID int64, media tg.InputMediaClass, caption string, replyTo int32, opts ...*params.SendMessage) (*Message, error)
	BoundSendMediaGroup(ctx context.Context, chatID int64, media []tg.InputMediaClass, replyTo int32, opts ...*params.SendMediaGroup) ([]*Message, error)
	BoundSendPoll(ctx context.Context, chatID int64, question string, options []string, replyTo int32, opts ...*params.SendPoll) (*Message, error)
	BoundSendRich(ctx context.Context, chatID int64, rm tg.InputRichMessageClass, replyTo int32, opts ...*params.SendMessage) (*Message, error)
	BoundSendVenue(ctx context.Context, chatID int64, lat, lng float64, title, address string, replyTo int32, opts ...*params.SendVenue) (*Message, error)
	BoundStub(ctx context.Context, method string) error
	BoundUnpin(ctx context.Context, chatID int64, msgID int32, opts ...*params.PinMessage) error
	BoundVote(ctx context.Context, chatID int64, msgID int32, options [][]byte) error
}

// PreCheckoutQueryBinder is the minimal set of client operations required by PreCheckoutQuery bound methods. It is satisfied automatically by *telegram.Client.
type PreCheckoutQueryBinder interface {
	BoundAnswerPreCheckout(ctx context.Context, queryID int64, opts ...*params.AnswerPreCheckout) error
}

// ShippingQueryBinder is the minimal set of client operations required by ShippingQuery bound methods. It is satisfied automatically by *telegram.Client.
type ShippingQueryBinder interface {
	BoundAnswerShipping(ctx context.Context, queryID int64, opts ...*params.AnswerShipping) error
}

// StoryBinder is the minimal set of client operations required by Story bound methods. It is satisfied automatically by *telegram.Client.
type StoryBinder interface {
	BoundStoryDelete(ctx context.Context, peerID int64, storyID int32) error
	BoundStoryDownload(ctx context.Context, peerID int64, storyID int32, opts ...*params.Download) ([]byte, error)
	BoundStoryEditCaption(ctx context.Context, peerID int64, storyID int32, opts ...*params.EditCaption) (*Story, error)
	BoundStoryEditMedia(ctx context.Context, peerID int64, storyID int32, media tg.InputMediaClass) (*Story, error)
	BoundStoryEditPrivacy(ctx context.Context, peerID int64, storyID int32, opts ...*params.EditPrivacy) (*Story, error)
	BoundStoryForward(ctx context.Context, fromChatID int64, storyID int32, chatID int64, opts ...*params.StoryForward) (*Message, error)
	BoundStoryReact(ctx context.Context, peerID int64, storyID int32, opts ...*params.React) error
	BoundStoryRead(ctx context.Context, peerID int64, storyID int32) error
	BoundStoryReply(ctx context.Context, peerID int64, storyID int32, text string, opts ...*params.SendMessage) (*Message, error)
	BoundStoryReplyMedia(ctx context.Context, peerID int64, storyID int32, media tg.InputMediaClass, caption string, opts ...*params.SendMessage) (*Message, error)
	BoundStub(ctx context.Context, method string) error
}

// UserBinder is the minimal set of client operations required by User bound methods. It is satisfied automatically by *telegram.Client.
type UserBinder interface {
	BoundArchiveUser(ctx context.Context, chatID int64) error
	BoundBlock(ctx context.Context, userID int64) error
	BoundGetCommonChats(ctx context.Context, userID int64, limit int) ([]*Chat, error)
	BoundUnarchiveUser(ctx context.Context, chatID int64) error
	BoundUnblock(ctx context.Context, userID int64) error
}
