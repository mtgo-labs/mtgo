package types

import (
	"errors"

	"github.com/mtgo-labs/mtgo/telegram/params"
	"github.com/mtgo-labs/mtgo/tg"
)

var (
	// ErrNoBinder is returned when a bound method is called on a Message that was
	// not created by a client (e.g. constructed manually or from a cached state).
	ErrNoBinder = errors.New("types: bound methods not available (message was not created by a client)")
	// ErrNoChatBinder is returned when a bound method is called on a Chat that was
	// not created by a client.
	ErrNoChatBinder = errors.New("types: bound methods not available (chat was not created by a client)")
	// ErrNoInlineBinder is returned when a bound method is called on an InlineQuery
	// that was not created by a client.
	ErrNoInlineBinder = errors.New("types: bound methods not available (inline query was not created by a client)")
)

// Binder defines the set of message-level operations that a Client injects into
// a Message after creation. Each method corresponds to a single Telegram API call.
// Types embed a Binder so they can offer fluent, context-aware methods (Reply,
// Forward, Edit, etc.) without holding a direct reference to the Client.
//
// Example:
//
//	// The Binder is injected by the Client; users typically call the
//	// high-level Message methods instead of interacting with Binder directly.
//	reply, err := msg.Reply("Hello!")
type Binder interface {
	// BoundSend sends a text message to the given chat, optionally replying to
	// another message identified by replyTo.
	BoundSend(chatID int64, text string, replyTo int32, opts ...*params.SendMessage) (*Message, error)
	// BoundSendMedia uploads or re-sends media to the given chat with an optional
	// caption and reply-to reference.
	BoundSendMedia(chatID int64, media tg.InputMediaClass, caption string, replyTo int32, opts ...*params.SendMessage) (*Message, error)
	// BoundForward forwards a message from one chat to another.
	BoundForward(chatID int64, fromChatID int64, msgID int32, opts ...*params.ForwardMessages) (*Message, error)
	// BoundCopy copies a message into another chat without the forward header.
	BoundCopy(chatID int64, fromChatID int64, msgID int32, opts ...*params.CopyMessage) (int64, error)
	// BoundEdit modifies the text of an existing message.
	BoundEdit(chatID int64, msgID int32, text string, opts ...*params.EditMessage) (*Message, error)
	// BoundEditInline modifies the text of an inline message.
	BoundEditInline(inlineMessageID tg.InputBotInlineMessageIDClass, text string, opts ...*params.EditMessage) (bool, error)
	// BoundEditCaption changes the caption of a media message.
	BoundEditCaption(chatID int64, msgID int32, caption string, opts ...*params.EditMessage) (*Message, error)
	// BoundEditInlineCaption changes the caption of an inline media message.
	BoundEditInlineCaption(inlineMessageID tg.InputBotInlineMessageIDClass, caption string, opts ...*params.EditMessage) (bool, error)
	// BoundEditMedia replaces the media content of an existing message.
	BoundEditMedia(chatID int64, msgID int32, media tg.InputMediaClass) (*Message, error)
	// BoundEditInlineMedia replaces the media content of an inline message.
	BoundEditInlineMedia(inlineMessageID tg.InputBotInlineMessageIDClass, media tg.InputMediaClass) (bool, error)
	// BoundEditReplyMarkup changes only the inline keyboard of an existing message.
	BoundEditReplyMarkup(chatID int64, msgID int32, markup tg.ReplyMarkupClass) (*Message, error)
	// BoundEditInlineReplyMarkup changes only the inline keyboard of an inline message.
	BoundEditInlineReplyMarkup(inlineMessageID tg.InputBotInlineMessageIDClass, markup tg.ReplyMarkupClass) (bool, error)
	// BoundDelete removes one or more messages by their IDs.
	BoundDelete(chatID int64, msgIDs []int32, opts ...*params.DeleteMessages) (int, error)
	// BoundReact adds an emoji reaction to a message.
	BoundReact(chatID int64, msgID int32, opts ...*params.React) error
	// BoundPin pins a message in the chat.
	BoundPin(chatID int64, msgID int32, opts ...*params.PinMessage) error
	// BoundUnpin removes a message from the pinned list.
	BoundUnpin(chatID int64, msgID int32, opts ...*params.PinMessage) error
	// BoundRead marks messages as read up to the given message ID.
	BoundRead(chatID int64, msgID int32) error
	// BoundAnswerCallback sends an answer to a callback query originated from a
	// button press.
	BoundAnswerCallback(queryID int64, opts ...*params.AnswerCallback) error
	// BoundDownload downloads the media attached to a message and returns its raw
	// bytes.
	BoundDownload(chatID int64, msgID int32, opts ...*params.Download) ([]byte, error)
	// BoundDownloadTo downloads the media attached to a message to a file and
	// returns the absolute file path.
	BoundDownloadTo(chatID int64, msgID int32, fileName string, opts ...*params.Download) (string, error)
	// BoundSendContact sends a contact card (phone number + name) to a chat.
	BoundSendContact(chatID int64, phone, firstName, lastName string, replyTo int32, opts ...*params.SendContact) (*Message, error)
	// BoundSendLocation sends a geographic location point to a chat.
	BoundSendLocation(chatID int64, lat, lng float64, replyTo int32, opts ...*params.SendLocation) (*Message, error)
	// BoundSendVenue sends a named venue (location + title + address) to a chat.
	BoundSendVenue(chatID int64, lat, lng float64, title, address string, replyTo int32, opts ...*params.SendVenue) (*Message, error)
	// BoundSendPoll sends a poll question with the given answer options.
	BoundSendPoll(chatID int64, question string, options []string, replyTo int32, opts ...*params.SendPoll) (*Message, error)
	// BoundSendDice sends a dice (or other animated emoji) roll to a chat.
	BoundSendDice(chatID int64, emoji string, replyTo int32, opts ...*params.SendDice) (*Message, error)
	// BoundSendGame sends a game message using the game's short name.
	BoundSendGame(chatID int64, gameShortName string, replyTo int32, opts ...*params.SendGame) (*Message, error)
	// BoundSendMediaGroup sends an album of media items as a single grouped
	// message.
	BoundSendMediaGroup(chatID int64, media []tg.InputMediaClass, replyTo int32, opts ...*params.SendMediaGroup) ([]*Message, error)
	// BoundSendChatAction sends a "typing…" or other chat action indicator.
	BoundSendChatAction(chatID int64, action tg.SendMessageActionClass) error
	// BoundSendInlineBotResult sends a result chosen from an inline query to a
	// chat.
	BoundSendInlineBotResult(chatID int64, queryID int64, resultID string, replyTo int32, opts ...*params.SendInlineBotResult) (*Message, error)
	// BoundVote casts a vote in a poll using the selected option byte sequences.
	BoundVote(chatID int64, msgID int32, options [][]byte) error
	// BoundRetractVote retracts the user's vote in a poll.
	BoundRetractVote(chatID int64, msgID int32) error
	// BoundGetMediaGroup retrieves all messages that belong to the same album as
	// the given message.
	BoundGetMediaGroup(chatID int64, msgID int32) ([]*Message, error)
	// BoundCopyMediaGroup copies an entire media album into another chat.
	BoundCopyMediaGroup(chatID int64, fromChatID int64, msgID int32) ([]*Message, error)
	// BoundStub is a placeholder for methods that have not been implemented yet.
	// It always returns an error identifying the unimplemented method name.
	BoundStub(method string) error
	// BoundAnswerInline sends inline query results to answer an inline query.
	BoundAnswerInline(queryID int64, results []tg.InputBotInlineResultClass, opts ...*params.InlineQuery) error
	// BoundBlock blocks a user.
	BoundBlock(userID int64) error
	// BoundUnblock removes a user from the block list.
	BoundUnblock(userID int64) error
	// BoundGetCommonChats returns the list of common chats with a user.
	BoundGetCommonChats(userID int64, limit int) ([]*Chat, error)
	// BoundArchiveUser archives a user chat (moves to archived folder).
	BoundArchiveUser(chatID int64) error
	// BoundUnarchiveUser unarchives a user chat.
	BoundUnarchiveUser(chatID int64) error
	// BoundAnswerPreCheckout approves or rejects a pre-checkout query.
	BoundAnswerPreCheckout(queryID int64, opts ...*params.AnswerPreCheckout) error
	// BoundAnswerShipping returns shipping options or an error for a shipping query.
	BoundAnswerShipping(queryID int64, opts ...*params.AnswerShipping) error
	// BoundApproveJoinRequest approves a chat join request.
	BoundApproveJoinRequest(chatID int64, userID int64) error
	// BoundDeclineJoinRequest declines a chat join request.
	BoundDeclineJoinRequest(chatID int64, userID int64) error
	// BoundStoryReply sends a message as a reply to a story.
	BoundStoryReply(peerID int64, storyID int32, text string, opts ...*params.SendMessage) (*Message, error)
	// BoundStoryReplyMedia sends media as a reply to a story.
	BoundStoryReplyMedia(peerID int64, storyID int32, media tg.InputMediaClass, caption string, opts ...*params.SendMessage) (*Message, error)
	// BoundStoryForward forwards a story to another chat.
	BoundStoryForward(fromChatID int64, storyID int32, chatID int64, opts ...*params.StoryForward) (*Message, error)
	// BoundStoryRead marks a story as read.
	BoundStoryRead(peerID int64, storyID int32) error
	// BoundStoryDelete deletes a story.
	BoundStoryDelete(peerID int64, storyID int32) error
	// BoundStoryEditCaption edits the caption of a story.
	BoundStoryEditCaption(peerID int64, storyID int32, opts ...*params.EditCaption) (*Story, error)
	// BoundStoryEditMedia edits the media of a story.
	BoundStoryEditMedia(peerID int64, storyID int32, media tg.InputMediaClass) (*Story, error)
	// BoundStoryEditPrivacy edits the privacy of a story.
	BoundStoryEditPrivacy(peerID int64, storyID int32, opts ...*params.EditPrivacy) (*Story, error)
	// BoundStoryReact adds a reaction to a story.
	BoundStoryReact(peerID int64, storyID int32, opts ...*params.React) error
	// BoundStoryDownload downloads the media of a story.
	BoundStoryDownload(peerID int64, storyID int32, opts ...*params.Download) ([]byte, error)
	BoundReplyChecklist(chatID int64, checklist *tg.InputMediaTodo, replyTo int32, opts ...*params.SendChecklist) (*Message, error)
	BoundReplyPhoto(chatID int64, file *InputFile, caption string, replyTo int32, opts ...*params.SendPhoto) (*Message, error)
	BoundAnswerPhoto(chatID int64, file *InputFile, caption string, opts ...*params.SendPhoto) (*Message, error)
	BoundReplyAudio(chatID int64, file *InputFile, caption string, replyTo int32, opts ...*params.SendAudio) (*Message, error)
	BoundAnswerAudio(chatID int64, file *InputFile, caption string, opts ...*params.SendAudio) (*Message, error)
	BoundReplyVideo(chatID int64, file *InputFile, caption string, replyTo int32, opts ...*params.SendVideo) (*Message, error)
	BoundAnswerVideo(chatID int64, file *InputFile, caption string, opts ...*params.SendVideo) (*Message, error)
	BoundReplyDocument(chatID int64, file *InputFile, caption string, replyTo int32, opts ...*params.SendDocument) (*Message, error)
	BoundAnswerDocument(chatID int64, file *InputFile, caption string, opts ...*params.SendDocument) (*Message, error)
	BoundReplyAnimation(chatID int64, file *InputFile, caption string, replyTo int32, opts ...*params.SendAnimation) (*Message, error)
	BoundAnswerAnimation(chatID int64, file *InputFile, caption string, opts ...*params.SendAnimation) (*Message, error)
	BoundReplyVoice(chatID int64, file *InputFile, caption string, replyTo int32, opts ...*params.SendVoice) (*Message, error)
	BoundAnswerVoice(chatID int64, file *InputFile, caption string, opts ...*params.SendVoice) (*Message, error)
	BoundReplyVideoNote(chatID int64, file *InputFile, replyTo int32, opts ...*params.SendVideoNote) (*Message, error)
	BoundAnswerVideoNote(chatID int64, file *InputFile, opts ...*params.SendVideoNote) (*Message, error)
	BoundReplySticker(chatID int64, file *InputFile, replyTo int32, opts ...*params.SendSticker) (*Message, error)
	BoundAnswerSticker(chatID int64, file *InputFile, opts ...*params.SendSticker) (*Message, error)
}

// ChatBinder defines the set of chat-level operations that a Client injects into
// a Chat after creation. These methods allow a Chat object to perform actions
// like archiving, editing settings, managing members, and exporting invite links.
//
// Example:
//
//	// The ChatBinder is injected by the Client; users typically call the
//	// high-level Chat methods instead of interacting with ChatBinder directly.
//	err := chat.SetTitle("Updated Title")
type ChatBinder interface {
	// BoundArchive moves the chat to the archived folder.
	BoundArchive(chatID int64) error
	// BoundUnarchive moves the chat back to the main chat list.
	BoundUnarchive(chatID int64) error
	// BoundSetTitle changes the chat's display title.
	BoundSetTitle(chatID int64, title string) error
	// BoundSetDescription updates the chat's about/description text.
	BoundSetDescription(chatID int64, description string) error
	// BoundSetPhoto changes the chat's profile photo.
	BoundSetPhoto(chatID int64, photo tg.InputChatPhotoClass) error
	// BoundDeletePhoto removes the chat's profile photo.
	BoundDeletePhoto(chatID int64) error
	// BoundSetUsername changes the public username of the chat.
	BoundSetUsername(chatID int64, username string) error
	// BoundBanMember permanently or temporarily bans a user from the chat.
	BoundBanMember(chatID int64, userID int64) error
	// BoundUnbanMember removes a ban, allowing the user to rejoin.
	BoundUnbanMember(chatID int64, userID int64) error
	// BoundRestrictMember applies the given banned rights to a user, limiting
	// what they can do in the chat.
	BoundRestrictMember(chatID int64, userID int64, bannedRights *tg.ChatBannedRights) error
	// BoundPromoteMember grants the given admin rights to a user.
	BoundPromoteMember(chatID int64, userID int64, adminRights *tg.ChatAdminRights) error
	// BoundJoinChat joins a chat using its username.
	BoundJoinChat(chatID int64, username string) (*Chat, error)
	// BoundLeaveChat removes the current user from the chat.
	BoundLeaveChat(chatID int64) error
	// BoundExportInviteLink generates a new primary invite link for the chat.
	BoundExportInviteLink(chatID int64) (string, error)
	// BoundGetMember retrieves a single chat member by user ID.
	BoundGetMember(chatID int64, userID int64) (*ChatMember, error)
	// BoundGetMember retrieves a page of chat members with the given limit and
	// offset.
	BoundGetMembers(chatID int64, limit int, offset int) ([]*ChatMember, error)
	// BoundAddMembers adds a user to the chat.
	BoundAddMembers(chatID int64, userID int64) error
	// BoundMarkUnread toggles the unread marker on a chat.
	BoundMarkUnread(chatID int64, unread bool) error
	// BoundSetProtectedContent enables or disables content protection (no-forward
	// restriction) on the chat.
	BoundSetProtectedContent(chatID int64, enabled bool) error
	// BoundSetTTL sets the auto-delete TTL (Time-To-Live) for messages in the
	// chat, in seconds.
	BoundSetTTL(chatID int64, ttl int) error
	// BoundSetPermissions applies the given default permissions for non-admin
	// members.
	BoundSetPermissions(chatID int64, permissions *tg.ChatBannedRights) error
	// BoundSetAdminTitle assigns a custom title to an admin in a supergroup.
	BoundSetAdminTitle(chatID int64, userID int64, title string) error
	// BoundSetSlowMode enables slow mode with the given cooldown in seconds.
	// Pass 0 to disable.
	BoundSetSlowMode(chatID int64, seconds int) error
	// BoundMute disables notifications for the chat.
	BoundMute(chatID int64) error
	// BoundUnmute re-enables notifications for the chat.
	BoundUnmute(chatID int64) error
	// BoundUnpinAll removes all pinned messages from the chat and returns the
	// count of unpinned messages.
	BoundUnpinAll(chatID int64) (int, error)
	// BoundGetChat fetches the latest full chat information from the server.
	BoundGetChat(chatID int64) (*Chat, error)
	// BoundGetEventLog retrieves recent admin log events matching the query.
	BoundGetEventLog(chatID int64, query string, limit int) ([]*ChatEvent, error)
}
