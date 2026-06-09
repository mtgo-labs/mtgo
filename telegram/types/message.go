package types

import (
	"fmt"
	"strconv"
	"time"

	"github.com/mtgo-labs/mtgo/telegram/params"
	"github.com/mtgo-labs/mtgo/tg"
)

// Message represents a Telegram message, containing text, media, metadata,
// and optional service action information. When created by a Client, it carries
// a Binder so that Reply, Edit, Delete, and other convenience methods can
// operate directly against the API without the caller needing a Client reference.
//
// Example:
//
//	msg, err := client.SendMessage(ctx, chatID, "Hello!")
//	fmt.Printf("Sent message ID: %d, Text: %s\n", msg.ID, msg.Text)
type Message struct {
	// ID is the unique message identifier within the chat.
	ID int32
	// Date is when the message was sent.
	Date time.Time
	// Text is the textual content of the message, empty for media-only or service
	// messages.
	Text string
	// FromID is the ID of the sender. Negative for chats/channels, positive for
	// users.
	FromID int64
	// Sender is the resolved user who sent the message, or nil if the sender
	// was not available in the peer map (e.g. anonymous channel admin).
	Sender *User
	// FromUser is an alias for Sender matching Pyrogram naming.
	FromUser *User
	// SenderChat is the resolved chat that sent the message on behalf of itself.
	SenderChat *Chat
	// SenderBoostCount is the number of boosts applied by the sender.
	SenderBoostCount int32
	// SenderBusinessBot is the business bot that sent this outgoing message.
	SenderBusinessBot *User
	// SenderTag is the sender rank or custom title in supergroups.
	SenderTag string
	// ChatID is the ID of the chat where the message was sent. Negative for groups
	// and channels.
	ChatID int64
	// Chat is the resolved chat where the message was sent, or nil if the chat
	// was not available in the peer map.
	Chat *Chat
	// AutomaticForward is true for automatically forwarded channel posts.
	AutomaticForward bool
	// FromOffline is true when sent by an implicit offline action.
	FromOffline bool
	// ReplyToID is the ID of the message this message replies to, or 0 if none.
	ReplyToID int32
	// ReplyToMessageID is an alias for ReplyToID matching Pyrogram naming.
	ReplyToMessageID int32
	// ReplyToTopMessageID is the first message that started the thread.
	ReplyToTopMessageID int32
	// ReplyToChecklistTaskID is the checklist task ID being replied to.
	ReplyToChecklistTaskID int32
	// ReplyToPollOptionID is the poll option ID being replied to.
	ReplyToPollOptionID string
	// Media holds the parsed media attachment, or nil for text-only messages.
	Media Media
	// Entities contains formatting and special entity markers (mentions, URLs,
	// bold, etc.) found in Text.
	Entities []*MessageEntity
	// ReplyMarkup is the inline or reply keyboard attached to the message.
	ReplyMarkup *ReplyMarkup
	// Out is true when the message was sent by the current user.
	Out bool
	// Outgoing is an alias for Out matching Pyrogram naming.
	Outgoing bool
	// Empty is true for messageEmpty objects.
	Empty bool
	// Mentioned is true when the current user was mentioned in the message.
	Mentioned bool
	// Silent is true when the message was sent without triggering a notification.
	Silent bool
	// Pinned is true when the message is currently pinned in the chat.
	Pinned bool
	// Views is the view count for channel posts; 0 when not applicable.
	Views int
	// Forwards is the number of times this message has been forwarded.
	Forwards int
	// EditDate is the timestamp of the last edit, or zero if never edited.
	EditDate time.Time
	// EditHidden is true when the edit is hidden by Telegram.
	EditHidden bool
	// PostAuthor is the author name displayed on channel posts when signatures
	// are enabled.
	PostAuthor string
	// AuthorSignature is an alias for PostAuthor matching Pyrogram naming.
	AuthorSignature string
	// GroupedID identifies messages that belong to the same album (grouped media).
	// 0 when not part of a group.
	GroupedID int64
	// MediaGroupID is an alias for GroupedID matching Pyrogram naming.
	MediaGroupID int64
	// ViaBotID is the user ID of the inline bot that generated this message, or 0
	// if not from a bot.
	ViaBotID int64
	// ViaBot is the resolved inline bot that generated this message.
	ViaBot *User
	// FwdFrom contains information about the original source when the message was
	// forwarded.
	FwdFrom *ForwardHeader
	// ForwardOrigin describes the Pyrogram-style origin of a forwarded message.
	ForwardOrigin *MessageOrigin
	// Reactions lists the emoji reactions currently attached to the message.
	Reactions []Reaction
	// TTLPeriod is the auto-delete timer in seconds; 0 means no auto-delete.
	TTLPeriod int
	// EffectID is the unique message effect identifier.
	EffectID int64
	// IsPaidPost is true when this is a paid suggested post.
	IsPaidPost bool
	// HasProtectedContent is true when forwarding is disabled.
	HasProtectedContent bool
	// VideoProcessingPending is true while video media is still processing.
	VideoProcessingPending bool
	// SendPaidMessagesStars is the number of stars paid to send the message.
	SendPaidMessagesStars int64
	// RepeatPeriod is the repeat period in seconds for scheduled messages.
	RepeatPeriod int32
	// SummaryLanguageCode is the language code available for message summaries.
	SummaryLanguageCode                     string
	GuestQueryID                            string
	TopicMessage                            bool
	Topic                                   *ForumTopic
	MessageThreadID                         int32
	DirectMessagesTopicID                   int64
	ReplyToStoryID                          int32
	ReplyToStoryUserID                      int64
	ReplyToMessage                          *Message
	ReplyToStory                            *Story
	PaidMedia                               *PaidMedia
	Checklist                               *Checklist
	ShowCaptionAboveMedia                   bool
	HasMediaSpoiler                         bool
	Caption                                 string
	CaptionEntities                         []*MessageEntity
	Audio                                   *DocumentMedia
	Document                                *DocumentMedia
	Photo                                   *Photo
	LivePhoto                               *LivePhoto
	Sticker                                 *Sticker
	Animation                               *Animation
	Game                                    *GameMedia
	Giveaway                                *GiveawayMedia
	Invoice                                 *InvoiceMedia
	Story                                   *StoryMedia
	Video                                   *DocumentMedia
	Voice                                   *DocumentMedia
	VideoNote                               *DocumentMedia
	Contact                                 *ContactMedia
	Location                                *LocationMedia
	Venue                                   *VenueMedia
	WebPage                                 *WebPageMedia
	LinkPreviewOptions                      *LinkPreviewOptions
	Poll                                    *PollMedia
	Dice                                    *DiceMedia
	UnreadMedia                             bool
	Legacy                                  bool
	RestrictionReason                       []*Restriction
	ExternalReply                           *ExternalReplyInfo
	Quote                                   *TextQuote
	Matches                                 []string
	Command                                 []string
	FactCheck                               *FactCheck
	SuggestedPostInfo                       *SuggestedPostInfo
	ChannelPost                             bool
	GuestBotCallerUser                      *User
	GuestBotCallerChat                      *Chat
	Link                                    string
	Content                                 string
	NewChatMembers                          []*User
	LeftChatMember                          *User
	ChatOwnerLeft                           *ChatOwnerLeft
	ChatOwnerChanged                        *ChatOwnerChanged
	ChatJoinType                            ChatJoinType
	NewChatTitle                            string
	NewChatPhoto                            *Photo
	DeleteChatPhoto                         bool
	GroupChatCreated                        bool
	SupergroupChatCreated                   bool
	ChannelChatCreated                      bool
	MigrateToChatID                         int64
	MigrateFromChatID                       int64
	PinnedMessage                           *Message
	GameHighScore                           *GameHighScore
	ForumTopicCreated                       *ForumTopicCreated
	ForumTopicClosed                        *ForumTopicClosed
	ForumTopicReopened                      *ForumTopicReopened
	ForumTopicEdited                        *ForumTopicEdited
	GeneralForumTopicHidden                 *GeneralForumTopicHidden
	GeneralForumTopicUnhidden               *GeneralForumTopicUnhidden
	VideoChatScheduled                      *VideoChatScheduled
	HistoryCleared                          bool
	VideoChatStarted                        *VideoChatStarted
	VideoChatEnded                          *VideoChatEnded
	VideoChatMembersInvited                 *VideoChatMembersInvited
	PhoneCallStarted                        *PhoneCallStarted
	PhoneCallEnded                          *PhoneCallEnded
	WebAppData                              *WebAppData
	PaidMessagesRefunded                    *PaidMessagesRefunded
	PaidMessagesPriceChanged                *PaidMessagesPriceChanged
	DirectMessagePriceChanged               *DirectMessagePriceChanged
	ChecklistTasksDone                      *ChecklistTasksDone
	ChecklistTasksAdded                     *ChecklistTasksAdded
	PremiumGiftCode                         *PremiumGiftCode
	GiftedPremium                           *GiftedPremium
	GiftedStars                             *GiftedStars
	GiftedTon                               *GiftedTon
	Gift                                    *Gift
	IsPrepaidUpgrade                        bool
	IsFromAuction                           bool
	SuggestProfilePhoto                     *Photo
	SuggestBirthday                         *Birthday
	UsersShared                             *UsersShared
	ChatShared                              *ChatShared
	SuccessfulPayment                       *SuccessfulPayment
	RefundedPayment                         *RefundedPayment
	SuggestedPostApprovalFailed             *SuggestedPostApprovalFailed
	SuggestedPostApproved                   *SuggestedPostApproved
	SuggestedPostDeclined                   *SuggestedPostDeclined
	SuggestedPostPaid                       *SuggestedPostPaid
	SuggestedPostRefunded                   *SuggestedPostRefunded
	GiveawayCreated                         *GiveawayCreated
	GiveawayWinners                         *GiveawayWinners
	GiveawayCompleted                       *GiveawayCompleted
	ManagedBotCreated                       *ManagedBotCreated
	PollOptionAdded                         *PollOptionAdded
	PollOptionDeleted                       *PollOptionDeleted
	ChatSetTheme                            *ChatTheme
	ChatSetBackground                       *ChatBackground
	SetMessageAutoDeleteTime                int32
	ChatBoost                               int32
	WriteAccessAllowed                      *WriteAccessAllowed
	ConnectedWebsite                        string
	ContactRegistered                       *ContactRegistered
	ProximityAlertTriggered                 *ProximityAlertTriggered
	GiveawayPrizeStars                      *GiveawayPrizeStars
	ScreenshotTaken                         *ScreenshotTaken
	UpgradedGiftPurchaseOffer               *UpgradedGiftPurchaseOffer
	UpgradedGiftPurchaseOfferRejected       *UpgradedGiftPurchaseOfferRejected
	ChatHasProtectedContentToggled          *ChatHasProtectedContentToggled
	ChatHasProtectedContentDisableRequested *ChatHasProtectedContentDisableRequested
	// Service is non-nil when the message is a service/system message (e.g. group
	// created, member added).
	Service *ServiceMessage
	// SenderChatID is the ID of the chat that sent the message on behalf of itself
	// (used for anonymous admin messages in groups).
	SenderChatID int64
	// TopicID is the forum topic ID this message belongs to, or 0 if not in a
	// forum topic.
	TopicID int32
	// IsFromPending is true when the message is a local placeholder not yet
	// confirmed by the server.
	IsFromPending bool
	// Raw is the original MTProto MessageClass (*tg.Message or *tg.MessageService),
	// preserved for advanced use cases that need access to the full TL object.
	Raw                  tg.MessageClass
	BusinessConnectionID string
	binder               Binder
	translate            func(key string, args ...any) string
}

// TranslatorFunc is the signature for a message-local translation function used
// to localize user-facing text via Message.T.
type TranslatorFunc func(key string, args ...any) string

func (m *Message) SetTranslator(fn TranslatorFunc) {
	m.translate = fn
}

func (m *Message) T(key string, args ...any) string {
	if m.translate == nil {
		return key
	}
	return m.translate(key, args...)
}

// ForwardHeader contains information about the original source of a forwarded message.
type ForwardHeader struct {
	// Date is when the original message was sent.
	Date time.Time
	// FromID is the user ID of the original sender.
	FromID int64
	// FromName is the display name of the original sender when the sender's account
	// is hidden.
	FromName string
	// ChannelID is the channel ID when the forward originates from a channel post.
	ChannelID int64
	// PostID is the message ID within the source channel when forwarded from a post.
	PostID int32
}

// Reaction represents a single emoji reaction attached to a message with its count.
type Reaction struct {
	Emoji         string
	CustomEmojiID string
	Count         int
	ChosenOrder   int
	IsPaid        bool
}

// ParseReaction converts a TL ReactionClass into a Reaction.
// Returns nil if raw is nil.
//
// Example:
//
//	react := types.ParseReaction(rawReaction)
//	if react != nil {
//	    fmt.Printf("Emoji: %s, count: %d\n", react.Emoji, react.Count)
//	}
func ParseReaction(raw tg.ReactionClass) *Reaction {
	if raw == nil {
		return nil
	}
	switch v := raw.(type) {
	case *tg.ReactionEmoji:
		return &Reaction{Emoji: v.Emoticon}
	case *tg.ReactionCustomEmoji:
		return &Reaction{CustomEmojiID: fmt.Sprintf("%d", v.DocumentID)}
	case *tg.ReactionPaid:
		return &Reaction{IsPaid: true}
	}
	return nil
}

func reactionToTL(r Reaction) tg.ReactionClass {
	if r.CustomEmojiID != "" {
		id, _ := strconv.ParseInt(r.CustomEmojiID, 10, 64)
		return &tg.ReactionCustomEmoji{DocumentID: id}
	}
	return &tg.ReactionEmoji{Emoticon: r.Emoji}
}

func ReactionsToTL(reactions []Reaction) []tg.ReactionClass {
	out := make([]tg.ReactionClass, len(reactions))
	for i, r := range reactions {
		out[i] = reactionToTL(r)
	}
	return out
}

// ServiceMessage wraps a service action type for system-generated messages such
// as group creation, member changes, or pinned message notifications.
type ServiceMessage struct {
	// Type identifies the specific kind of service action.
	Type ServiceActionType
	// RequestedPeers holds the shared peers when Type is ServiceActionRequestedPeer.
	// Each entry has a ButtonID matching the button that triggered the share, and
	// a list of peer IDs (positive for users, negative for chats, -100-prefixed
	// for channels and supergroups).
	RequestedPeers *RequestedPeerData
}

// RequestedPeerData holds the peer IDs returned by a keyboardButtonRequestPeer.
type RequestedPeerData struct {
	ButtonID int32
	// UserIDs contains the shared user IDs (empty for chat/channel shares).
	UserIDs []int64
	// ChatIDs contains the shared chat/channel IDs (negative, empty for user shares).
	ChatIDs []int64
}

const zeroChannelID = -1000000000000

type peerID int64

func (id *peerID) Channel(p int64) {
	*id = peerID(zeroChannelID - p)
}

// ServiceActionType enumerates the kinds of service actions that can appear in a message.
type ServiceActionType int

const (
	// ServiceActionGroupCreate indicates the group was created.
	ServiceActionGroupCreate ServiceActionType = iota
	// ServiceActionGroupEditTitle indicates the group title was changed.
	ServiceActionGroupEditTitle
	// ServiceActionGroupEditPhoto indicates the group photo was changed.
	ServiceActionGroupEditPhoto
	// ServiceActionGroupDeletePhoto indicates the group photo was removed.
	ServiceActionGroupDeletePhoto
	// ServiceActionGroupAddMembers indicates one or more members were added.
	ServiceActionGroupAddMembers
	// ServiceActionGroupRemoveMember indicates a member was removed.
	ServiceActionGroupRemoveMember
	// ServiceActionGroupJoinedByLink indicates a user joined via an invite link.
	ServiceActionGroupJoinedByLink
	// ServiceActionChannelCreate indicates a channel was created.
	ServiceActionChannelCreate
	// ServiceActionGroupMigrateTo indicates the group was upgraded to a
	// supergroup.
	ServiceActionGroupMigrateTo
	// ServiceActionChannelMigrateFrom indicates the supergroup was created from
	// a basic group.
	ServiceActionChannelMigrateFrom
	// ServiceActionPinMessage indicates a message was pinned.
	ServiceActionPinMessage
	// ServiceActionHistoryClear indicates the chat history was cleared.
	ServiceActionHistoryClear
	// ServiceActionGameScore indicates a game score was updated.
	ServiceActionGameScore
	// ServiceActionPhoneCall indicates a phone call event.
	ServiceActionPhoneCall
	// ServiceActionScreenshotTaken indicates a user took a screenshot in a
	// secret chat.
	ServiceActionScreenshotTaken
	// ServiceActionContactSignUp indicates a contact registered on Telegram.
	ServiceActionContactSignUp
	// ServiceActionGroupCall indicates a group voice chat event.
	ServiceActionGroupCall
	// ServiceActionSetTTL indicates the message TTL was changed.
	ServiceActionSetTTL
	// ServiceActionTopicCreate indicates a forum topic was created.
	ServiceActionTopicCreate
	// ServiceActionTopicEdit indicates a forum topic was edited.
	ServiceActionTopicEdit
	// ServiceActionGiftPremium indicates a Telegram Premium gift was sent.
	ServiceActionGiftPremium
	// ServiceActionBoostApply indicates a boost was applied to the chat.
	ServiceActionBoostApply
	// ServiceActionRequestedPeer indicates the user shared a chat/user via
	// a keyboardButtonRequestPeer button.
	ServiceActionRequestedPeer
	// ServiceActionUnknown represents an unrecognized service action.
	ServiceActionUnknown
)

// String returns a human-readable representation of the Message.
// It returns a truncated text preview, a media type label, or a fallback ID string.
func (m *Message) String() string {
	if m.Text != "" {
		if len(m.Text) > 50 {
			return m.Text[:50] + "..."
		}
		return m.Text
	}
	if m.Media != nil {
		return fmt.Sprintf("[%s]", m.Media.MediaType())
	}
	return fmt.Sprintf("msg_%d", m.ID)
}

// ParseMessage converts a raw MTProto MessageClass into a Message.
// It dispatches to the appropriate parser based on whether the message
// is a regular message, a service message, or empty.
// The PeerMap is used to resolve peer IDs. Returns nil if raw is nil.
//
// Example:
//
//	msg := types.ParseMessage(update.Message, peerMap)
//	if msg != nil {
//	    fmt.Println(msg.Text)
//	}
func ParseMessage(raw tg.MessageClass, pm *PeerMap) *Message {
	if raw == nil {
		return nil
	}
	switch r := raw.(type) {
	case *tg.MessageEmpty:
		m := &Message{
			ID:    r.ID,
			Empty: true,
			Raw:   r,
		}
		if r.PeerID != nil {
			m.ChatID = getBarePeerID(r.PeerID)
			m.Chat = ParseChatFromPeer(r.PeerID, pm)
		}
		return m
	case *tg.Message:
		return parseRegularMessage(r, pm)
	case *tg.MessageService:
		return parseServiceMessage(r, pm)
	}
	return nil
}

func parseRegularMessage(raw *tg.Message, pm *PeerMap) *Message {
	m := &Message{
		ID:        raw.ID,
		Date:      time.Unix(int64(raw.Date), 0),
		Text:      raw.Message,
		Out:       raw.Out,
		Outgoing:  raw.Out,
		Mentioned: raw.Mentioned,
		Silent:    raw.Silent,
		Pinned:    raw.Pinned,
		Raw:       raw,
	}
	if raw.FromID != nil {
		m.FromID = getPeerID(raw.FromID)
	}
	if m.FromID == 0 && !raw.Out && raw.PeerID != nil {
		if _, ok := raw.PeerID.(*tg.PeerUser); ok {
			m.FromID = getPeerID(raw.PeerID)
		}
	}
	if pm != nil && m.FromID > 0 {
		if u, ok := pm.Users[m.FromID]; ok {
			m.Sender = parseUserTL(u)
			m.FromUser = m.Sender
		}
	}
	if pm != nil && m.FromID < 0 && raw.FromID != nil {
		m.SenderChatID = m.FromID
		m.SenderChat = ParseChatFromPeer(raw.FromID, pm)
	}
	if raw.PeerID != nil {
		m.ChatID = getBarePeerID(raw.PeerID)
		m.Chat = ParseChatFromPeer(raw.PeerID, pm)
	}
	if raw.ReplyTo != nil {
		if rt, ok := raw.ReplyTo.(*tg.MessageReplyHeader); ok {
			m.TopicMessage = rt.ForumTopic
			if rt.ReplyToMsgID != 0 {
				m.ReplyToID = rt.ReplyToMsgID
				m.ReplyToMessageID = rt.ReplyToMsgID
			}
			if rt.ReplyToTopID != 0 {
				m.ReplyToTopMessageID = rt.ReplyToTopID
				m.MessageThreadID = rt.ReplyToTopID
			}
			if rt.TodoItemID != 0 {
				m.ReplyToChecklistTaskID = rt.TodoItemID
			}
			if rt.PollOption != nil {
				m.ReplyToPollOptionID = string(rt.PollOption)
			}
			m.ExternalReply = ParseExternalReplyInfo(rt, pm)
			m.Quote = parseTextQuote(rt)
		}
		if rt, ok := raw.ReplyTo.(*tg.MessageReplyStoryHeader); ok {
			m.ReplyToStoryID = rt.StoryID
			m.ReplyToStoryUserID = getPeerID(rt.Peer)
		}
	}
	if raw.Media != nil {
		m.Media = ParseMedia(raw.Media)
		m.LinkPreviewOptions = parseLinkPreviewOptions(raw.Media)
		m.setDirectMediaFields(raw.Media)
		if raw.Message != "" {
			m.Caption = raw.Message
		}
	}
	if raw.Entities != nil {
		var users map[int64]*tg.User
		if pm != nil {
			users = pm.Users
		}
		m.Entities = ParseMessageEntitiesWithUsers(raw.Entities, users)
		if raw.Media != nil {
			m.CaptionEntities = m.Entities
		}
	}
	if raw.ReplyMarkup != nil {
		m.ReplyMarkup = ParseReplyMarkup(raw.ReplyMarkup)
	}
	if raw.Views != 0 {
		m.Views = int(raw.Views)
	}
	if raw.Forwards != 0 {
		m.Forwards = int(raw.Forwards)
	}
	if raw.EditDate != 0 {
		m.EditDate = time.Unix(int64(raw.EditDate), 0)
	}
	m.EditHidden = raw.EditHide
	if raw.PostAuthor != "" {
		m.PostAuthor = raw.PostAuthor
		m.AuthorSignature = raw.PostAuthor
	}
	if raw.GroupedID != 0 {
		m.GroupedID = raw.GroupedID
		m.MediaGroupID = raw.GroupedID
	}
	if raw.ViaBotID != 0 {
		m.ViaBotID = raw.ViaBotID
		if pm != nil {
			if u, ok := pm.Users[raw.ViaBotID]; ok {
				m.ViaBot = parseUserTL(u)
			}
		}
	}
	if raw.ViaBusinessBotID != 0 && pm != nil {
		if u, ok := pm.Users[raw.ViaBusinessBotID]; ok {
			m.SenderBusinessBot = parseUserTL(u)
		}
	}
	if raw.FwdFrom != nil {
		m.FwdFrom = parseForwardHeader(raw.FwdFrom)
		m.ForwardOrigin = parseMessageOrigin(raw.FwdFrom, pm)
	}
	if raw.Reactions != nil && raw.Reactions.Results != nil {
		for _, r := range raw.Reactions.Results {
			if r != nil {
				react := Reaction{Count: int(r.Count)}
				if r.Reaction != nil {
					if er, ok := r.Reaction.(*tg.ReactionEmoji); ok {
						react.Emoji = er.Emoticon
					}
					if cr, ok := r.Reaction.(*tg.ReactionCustomEmoji); ok {
						react.CustomEmojiID = fmt.Sprintf("%d", cr.DocumentID)
					}
					if _, ok := r.Reaction.(*tg.ReactionPaid); ok {
						react.IsPaid = true
					}
				}
				m.Reactions = append(m.Reactions, react)
			}
		}
	}
	if raw.TTLPeriod != 0 {
		m.TTLPeriod = int(raw.TTLPeriod)
	}
	if raw.FromBoostsApplied != 0 {
		m.SenderBoostCount = raw.FromBoostsApplied
	}
	if raw.FromRank != "" {
		m.SenderTag = raw.FromRank
	}
	m.FromOffline = raw.Offline
	m.HasProtectedContent = raw.Noforwards
	m.VideoProcessingPending = raw.VideoProcessingPending
	m.IsPaidPost = raw.PaidSuggestedPostStars || raw.PaidSuggestedPostTon
	if raw.Effect != 0 {
		m.EffectID = raw.Effect
	}
	if raw.PaidMessageStars != 0 {
		m.SendPaidMessagesStars = raw.PaidMessageStars
	}
	if raw.ScheduleRepeatPeriod != 0 {
		m.RepeatPeriod = raw.ScheduleRepeatPeriod
	}
	if raw.SummaryFromLanguage != "" {
		m.SummaryLanguageCode = raw.SummaryFromLanguage
	}
	if raw.RestrictionReason != nil {
		m.RestrictionReason = parseRestrictions(raw.RestrictionReason)
	}
	if raw.Factcheck != nil {
		m.FactCheck = ParseFactCheck(raw.Factcheck)
	}
	if raw.SuggestedPost != nil {
		m.SuggestedPostInfo = parseSuggestedPostInfo(raw.SuggestedPost)
	}
	m.Content = m.Text
	if m.Caption != "" {
		m.Content = m.Caption
	}
	m.Link = buildMessageLink(m.Chat, m.ID)
	return m
}

func parseServiceMessage(raw *tg.MessageService, pm *PeerMap) *Message {
	m := &Message{
		ID:       raw.ID,
		Date:     time.Unix(int64(raw.Date), 0),
		Out:      raw.Out,
		Outgoing: raw.Out,
		Silent:   raw.Silent,
		Raw:      raw,
	}
	if raw.FromID != nil {
		m.FromID = getPeerID(raw.FromID)
	}
	if m.FromID == 0 && !raw.Out && raw.PeerID != nil {
		if _, ok := raw.PeerID.(*tg.PeerUser); ok {
			m.FromID = getPeerID(raw.PeerID)
		}
	}
	if pm != nil && m.FromID > 0 {
		if u, ok := pm.Users[m.FromID]; ok {
			m.Sender = parseUserTL(u)
			m.FromUser = m.Sender
		}
	}
	if pm != nil && m.FromID < 0 && raw.FromID != nil {
		m.SenderChatID = m.FromID
		m.SenderChat = ParseChatFromPeer(raw.FromID, pm)
	}
	if raw.PeerID != nil {
		m.ChatID = getBarePeerID(raw.PeerID)
		m.Chat = ParseChatFromPeer(raw.PeerID, pm)
	}
	if raw.ReplyTo != nil {
		if rt, ok := raw.ReplyTo.(*tg.MessageReplyHeader); ok {
			m.TopicMessage = rt.ForumTopic
			if rt.ReplyToMsgID != 0 {
				m.ReplyToID = rt.ReplyToMsgID
				m.ReplyToMessageID = rt.ReplyToMsgID
			}
			if rt.ReplyToTopID != 0 {
				m.ReplyToTopMessageID = rt.ReplyToTopID
				m.MessageThreadID = rt.ReplyToTopID
			}
			if rt.TodoItemID != 0 {
				m.ReplyToChecklistTaskID = rt.TodoItemID
			}
			if rt.PollOption != nil {
				m.ReplyToPollOptionID = string(rt.PollOption)
			}
			m.ExternalReply = ParseExternalReplyInfo(rt, pm)
			m.Quote = parseTextQuote(rt)
		}
		if rt, ok := raw.ReplyTo.(*tg.MessageReplyStoryHeader); ok {
			m.ReplyToStoryID = rt.StoryID
			m.ReplyToStoryUserID = getPeerID(rt.Peer)
		}
	}
	if raw.Action != nil {
		m.Service = parseServiceAction(raw.Action)
		m.setDirectServiceFields(raw.Action, pm)
	}
	m.Link = buildMessageLink(m.Chat, m.ID)
	return m
}

func (m *Message) setDirectMediaFields(raw tg.MessageMediaClass) {
	if m.Media != nil {
		switch media := m.Media.(type) {
		case *PhotoMedia:
			m.Photo = media.Photo
			m.HasMediaSpoiler = media.IsSpoiler
		case *DocumentMedia:
			m.Document = media
			m.HasMediaSpoiler = media.IsSpoiler
			switch media.MediaType() {
			case MessageMediaTypeAnimation:
				m.Animation = parseAnimationFromDoc(media)
			case MessageMediaTypeAudio:
				m.Audio = media
			case MessageMediaTypeSticker:
				m.Sticker = parseStickerFromDoc(media)
			case MessageMediaTypeVideo:
				m.Video = media
			case MessageMediaTypeVideoNote:
				m.VideoNote = media
			case MessageMediaTypeVoice:
				m.Voice = media
			}
		case *WebPageMedia:
			m.WebPage = media
		case *PollMedia:
			m.Poll = media
		case *DiceMedia:
			m.Dice = media
		case *GameMedia:
			m.Game = media
		case *InvoiceMedia:
			m.Invoice = media
		case *StoryMedia:
			m.Story = media
		case *GiveawayMedia:
			m.Giveaway = media
		case *PaidMedia:
			m.PaidMedia = media
		case *ContactMedia:
			m.Contact = media
		case *LocationMedia:
			m.Location = media
		case *VenueMedia:
			m.Venue = media
		}
	}
	if todo, ok := raw.(*tg.MessageMediaToDo); ok {
		m.Checklist = ParseChecklist(todo)
	}
}

func (m *Message) setDirectServiceFields(raw tg.MessageActionClass, pm *PeerMap) {
	switch action := raw.(type) {
	case *tg.MessageActionChatCreate:
		m.GroupChatCreated = true
		m.NewChatTitle = action.Title
		m.NewChatMembers = usersFromIDs(action.Users, pm)
	case *tg.MessageActionChatEditTitle:
		m.NewChatTitle = action.Title
	case *tg.MessageActionChatEditPhoto:
		m.NewChatPhoto = photoFromClass(action.Photo)
	case *tg.MessageActionChatDeletePhoto:
		m.DeleteChatPhoto = true
	case *tg.MessageActionChatAddUser:
		m.NewChatMembers = usersFromIDs(action.Users, pm)
	case *tg.MessageActionChatDeleteUser:
		m.LeftChatMember = userFromID(action.UserID, pm)
	case *tg.MessageActionChannelCreate:
		m.ChannelChatCreated = true
		m.NewChatTitle = action.Title
	case *tg.MessageActionChatMigrateTo:
		m.MigrateToChatID = channelChatID(action.ChannelID)
	case *tg.MessageActionChannelMigrateFrom:
		m.SupergroupChatCreated = true
		m.MigrateFromChatID = -action.ChatID
	case *tg.MessageActionPinMessage:
		m.Pinned = true
	case *tg.MessageActionHistoryClear:
		m.HistoryCleared = true
	case *tg.MessageActionPhoneCall:
		if action.Duration != 0 || action.Reason != nil {
			m.PhoneCallEnded = ParsePhoneCallEnded(action)
		} else {
			m.PhoneCallStarted = ParsePhoneCallStarted(action)
		}
	case *tg.MessageActionScreenshotTaken:
		m.ScreenshotTaken = &ScreenshotTaken{}
	case *tg.MessageActionContactSignUp:
		m.ContactRegistered = &ContactRegistered{}
	case *tg.MessageActionGeoProximityReached:
		m.ProximityAlertTriggered = ParseProximityAlertTriggered(action)
	case *tg.MessageActionGroupCall:
		if action.Duration != 0 {
			m.VideoChatEnded = ParseVideoChatEnded(action)
		} else {
			m.VideoChatStarted = ParseVideoChatStarted(action)
		}
	case *tg.MessageActionInviteToGroupCall:
		m.VideoChatMembersInvited = ParseVideoChatMembersInvited(action)
	case *tg.MessageActionSetMessagesTTL:
		m.SetMessageAutoDeleteTime = action.Period
	case *tg.MessageActionGroupCallScheduled:
		m.VideoChatScheduled = ParseVideoChatScheduled(action)
	case *tg.MessageActionWebViewDataSentMe:
		m.WebAppData = &WebAppData{Data: action.Data, ButtonText: action.Text}
	case *tg.MessageActionTopicCreate:
		m.ForumTopicCreated = &ForumTopicCreated{
			Title:         action.Title,
			IconColor:     action.IconColor,
			CustomEmojiID: fmt.Sprintf("%d", action.IconEmojiID),
		}
	case *tg.MessageActionTopicEdit:
		m.ForumTopicEdited = &ForumTopicEdited{
			Title:         action.Title,
			CustomEmojiID: fmt.Sprintf("%d", action.IconEmojiID),
			IsClosed:      action.Closed,
			IsHidden:      action.Hidden,
		}
		if action.Closed {
			m.ForumTopicClosed = &ForumTopicClosed{}
		}
		if action.Hidden {
			m.GeneralForumTopicHidden = &GeneralForumTopicHidden{}
		}
	case *tg.MessageActionBotAllowed:
		m.WriteAccessAllowed = ParseWriteAccessAllowed(action)
	case *tg.MessageActionSetChatWallPaper:
		m.ChatSetBackground = ParseChatBackground(action)
	case *tg.MessageActionPaidMessagesRefunded:
		m.PaidMessagesRefunded = &PaidMessagesRefunded{
			MessageCount: action.Count,
			StarCount:    action.Stars,
		}
	case *tg.MessageActionPaidMessagesPrice:
		m.PaidMessagesPriceChanged = &PaidMessagesPriceChanged{Stars: action.Stars}
	case *tg.MessageActionTodoCompletions:
		m.ChecklistTasksDone = &ChecklistTasksDone{
			MarkedAsDoneTaskIDs:    action.Completed,
			MarkedAsNotDoneTaskIDs: action.Incompleted,
		}
	case *tg.MessageActionTodoAppendTasks:
		m.ChecklistTasksAdded = &ChecklistTasksAdded{
			Tasks: checklistTasksFromTodoItems(action.List),
		}
	case *tg.MessageActionPaymentRefunded:
		m.RefundedPayment = &RefundedPayment{
			Currency:    action.Currency,
			TotalAmount: action.TotalAmount,
			Payload:     action.Payload,
		}
		if action.Charge != nil {
			m.RefundedPayment.ChargeID = action.Charge.ID
			m.RefundedPayment.ProviderChargeID = action.Charge.ProviderChargeID
		}
	case *tg.MessageActionManagedBotCreated:
		m.ManagedBotCreated = &ManagedBotCreated{Bot: userFromID(action.BotID, pm)}
	}
}

func photoFromClass(raw tg.PhotoClass) *Photo {
	if raw == nil {
		return nil
	}
	if p, ok := raw.(*tg.Photo); ok {
		return parsePhoto(p)
	}
	return nil
}

func userFromID(id int64, pm *PeerMap) *User {
	if id == 0 {
		return nil
	}
	if pm != nil {
		if u, ok := pm.Users[id]; ok {
			return parseUserTL(u)
		}
	}
	return &User{ID: id}
}

func usersFromIDs(ids []int64, pm *PeerMap) []*User {
	if ids == nil {
		return nil
	}
	users := make([]*User, 0, len(ids))
	for _, id := range ids {
		if user := userFromID(id, pm); user != nil {
			users = append(users, user)
		}
	}
	return users
}

func checklistTasksFromTodoItems(items []*tg.TodoItem) []*ChecklistTask {
	if items == nil {
		return nil
	}
	tasks := make([]*ChecklistTask, 0, len(items))
	for _, item := range items {
		if item == nil {
			continue
		}
		task := &ChecklistTask{ID: item.ID}
		if item.Title != nil {
			task.Text = item.Title.Text
			task.Entities = ParseMessageEntities(item.Title.Entities)
		}
		tasks = append(tasks, task)
	}
	return tasks
}

func parseTextQuote(raw *tg.MessageReplyHeader) *TextQuote {
	if raw == nil || raw.QuoteText == "" {
		return nil
	}
	return &TextQuote{
		Text:     raw.QuoteText,
		Entities: ParseMessageEntities(raw.QuoteEntities),
		Offset:   raw.QuoteOffset,
		Position: raw.QuoteOffset,
		IsManual: raw.Quote,
	}
}

func parseSuggestedPostInfo(raw *tg.SuggestedPost) *SuggestedPostInfo {
	if raw == nil {
		return nil
	}
	info := &SuggestedPostInfo{State: SuggestedPostStatePending}
	if raw.Accepted {
		info.State = SuggestedPostStateApproved
	}
	if raw.Rejected {
		info.State = SuggestedPostStateDeclined
	}
	if raw.ScheduleDate != 0 {
		info.SendDate = time.Unix(int64(raw.ScheduleDate), 0)
	}
	if raw.Price != nil {
		info.Price = suggestedPostPrice(raw.Price)
	}
	return info
}

func suggestedPostPrice(raw tg.StarsAmountClass) *SuggestedPostPrice {
	switch price := raw.(type) {
	case *tg.StarsAmount:
		return &SuggestedPostPrice{
			Amount:    price.Amount,
			Currency:  "XTR",
			StarCount: price.Amount,
		}
	case *tg.StarsTonAmount:
		return &SuggestedPostPrice{
			Amount:   price.Amount,
			Currency: "TON",
		}
	}
	return nil
}

func buildMessageLink(chat *Chat, messageID int32) string {
	if chat == nil || chat.Username == "" || messageID == 0 {
		return ""
	}
	return fmt.Sprintf("https://t.me/%s/%d", chat.Username, messageID)
}

func parseServiceAction(raw tg.MessageActionClass) *ServiceMessage {
	switch action := raw.(type) {
	case *tg.MessageActionChatCreate:
		return &ServiceMessage{Type: ServiceActionGroupCreate}
	case *tg.MessageActionChatEditTitle:
		return &ServiceMessage{Type: ServiceActionGroupEditTitle}
	case *tg.MessageActionChatEditPhoto:
		return &ServiceMessage{Type: ServiceActionGroupEditPhoto}
	case *tg.MessageActionChatDeletePhoto:
		return &ServiceMessage{Type: ServiceActionGroupDeletePhoto}
	case *tg.MessageActionChatAddUser:
		return &ServiceMessage{Type: ServiceActionGroupAddMembers}
	case *tg.MessageActionChatDeleteUser:
		return &ServiceMessage{Type: ServiceActionGroupRemoveMember}
	case *tg.MessageActionChatJoinedByLink:
		return &ServiceMessage{Type: ServiceActionGroupJoinedByLink}
	case *tg.MessageActionChannelCreate:
		return &ServiceMessage{Type: ServiceActionChannelCreate}
	case *tg.MessageActionChatMigrateTo:
		return &ServiceMessage{Type: ServiceActionGroupMigrateTo}
	case *tg.MessageActionChannelMigrateFrom:
		return &ServiceMessage{Type: ServiceActionChannelMigrateFrom}
	case *tg.MessageActionPinMessage:
		return &ServiceMessage{Type: ServiceActionPinMessage}
	case *tg.MessageActionHistoryClear:
		return &ServiceMessage{Type: ServiceActionHistoryClear}
	case *tg.MessageActionGameScore:
		return &ServiceMessage{Type: ServiceActionGameScore}
	case *tg.MessageActionPhoneCall:
		return &ServiceMessage{Type: ServiceActionPhoneCall}
	case *tg.MessageActionScreenshotTaken:
		return &ServiceMessage{Type: ServiceActionScreenshotTaken}
	case *tg.MessageActionContactSignUp:
		return &ServiceMessage{Type: ServiceActionContactSignUp}
	case *tg.MessageActionGroupCall:
		return &ServiceMessage{Type: ServiceActionGroupCall}
	case *tg.MessageActionSetMessagesTTL:
		return &ServiceMessage{Type: ServiceActionSetTTL}
	case *tg.MessageActionTopicCreate:
		return &ServiceMessage{Type: ServiceActionTopicCreate}
	case *tg.MessageActionTopicEdit:
		return &ServiceMessage{Type: ServiceActionTopicEdit}
	case *tg.MessageActionGiftPremium:
		return &ServiceMessage{Type: ServiceActionGiftPremium}
	case *tg.MessageActionBoostApply:
		return &ServiceMessage{Type: ServiceActionBoostApply}
	case *tg.MessageActionRequestedPeer:
		return &ServiceMessage{
			Type:           ServiceActionRequestedPeer,
			RequestedPeers: parseRequestedPeers(action.ButtonID, action.Peers),
		}
	case *tg.MessageActionRequestedPeerSentMe:
		return &ServiceMessage{
			Type:           ServiceActionRequestedPeer,
			RequestedPeers: parseRequestedPeerSentMe(action.ButtonID, action.Peers),
		}
	}
	return &ServiceMessage{Type: ServiceActionUnknown}
}

func parseRequestedPeers(buttonID int32, peers []tg.PeerClass) *RequestedPeerData {
	data := &RequestedPeerData{ButtonID: buttonID}
	for _, p := range peers {
		switch peer := p.(type) {
		case *tg.PeerUser:
			data.UserIDs = append(data.UserIDs, peer.UserID)
		case *tg.PeerChat:
			data.ChatIDs = append(data.ChatIDs, -peer.ChatID)
		case *tg.PeerChannel:
			data.ChatIDs = append(data.ChatIDs, channelChatID(peer.ChannelID))
		}
	}
	return data
}

func parseRequestedPeerSentMe(buttonID int32, peers []tg.RequestedPeerClass) *RequestedPeerData {
	data := &RequestedPeerData{ButtonID: buttonID}
	for _, p := range peers {
		switch peer := p.(type) {
		case *tg.RequestedPeerUser:
			data.UserIDs = append(data.UserIDs, peer.UserID)
		case *tg.RequestedPeerChat:
			data.ChatIDs = append(data.ChatIDs, -peer.ChatID)
		case *tg.RequestedPeerChannel:
			data.ChatIDs = append(data.ChatIDs, channelChatID(peer.ChannelID))
		}
	}
	return data
}

func channelChatID(channelID int64) int64 {
	var id peerID
	id.Channel(channelID)
	return int64(id)
}

func parseForwardHeader(raw *tg.MessageFwdHeader) *ForwardHeader {
	if raw == nil {
		return nil
	}
	h := &ForwardHeader{
		Date: time.Unix(int64(raw.Date), 0),
	}
	if raw.FromID != nil {
		h.FromID = getPeerID(raw.FromID)
		if channelID, ok := raw.FromID.(*tg.PeerChannel); ok {
			h.ChannelID = channelID.ChannelID
		}
	}
	if raw.FromName != "" {
		h.FromName = raw.FromName
	}
	if raw.ChannelPost != 0 {
		h.PostID = raw.ChannelPost
	}
	return h
}

func parseMessageOrigin(raw *tg.MessageFwdHeader, pm *PeerMap) *MessageOrigin {
	if raw == nil {
		return nil
	}
	origin := &MessageOrigin{
		Date:     time.Unix(int64(raw.Date), 0),
		Imported: raw.Imported,
	}
	if raw.Imported {
		origin.Type = MessageOriginTypeImport
		return origin
	}
	if raw.FromName != "" {
		origin.Type = MessageOriginTypeHiddenUser
		origin.SenderUserName = raw.FromName
		return origin
	}
	if raw.FromID == nil {
		return origin
	}

	switch peer := raw.FromID.(type) {
	case *tg.PeerUser:
		origin.Type = MessageOriginTypeUser
		origin.SenderUserID = peer.UserID
		if pm != nil {
			if user, ok := pm.Users[peer.UserID]; ok {
				origin.SenderUser = parseUserTL(user)
			}
		}
	case *tg.PeerChat:
		origin.Type = MessageOriginTypeChat
		origin.SenderChatID = peer.ChatID
		origin.AuthorSignature = raw.PostAuthor
		if pm != nil {
			origin.SenderChat = ParseChatFromPeer(peer, pm)
		}
	case *tg.PeerChannel:
		origin.Type = MessageOriginTypeChannel
		origin.ChatID = channelChatID(peer.ChannelID)
		origin.MessageID = raw.ChannelPost
		origin.AuthorSignature = raw.PostAuthor
		if pm != nil {
			origin.Chat = ParseChatFromPeer(peer, pm)
		}
	}
	return origin
}

func getPeerID(peer tg.PeerClass) int64 {
	if peer == nil {
		return 0
	}
	switch p := peer.(type) {
	case *tg.PeerUser:
		return p.UserID
	case *tg.PeerChat:
		return -p.ChatID
	case *tg.PeerChannel:
		return channelChatID(p.ChannelID)
	}
	return 0
}

func getBarePeerID(peer tg.PeerClass) int64 {
	if peer == nil {
		return 0
	}
	switch p := peer.(type) {
	case *tg.PeerUser:
		return p.UserID
	case *tg.PeerChat:
		return -p.ChatID
	case *tg.PeerChannel:
		return channelChatID(p.ChannelID)
	}
	return 0
}

// SetBinder injects the Binder that backs all bound convenience methods on this
// Message. Called internally by the Client after constructing a Message from an
// update.
func (m *Message) SetBinder(b Binder) {
	m.binder = b
}

// Reply sends a text message in the same chat, quoting this message as a reply.
// Returns ErrNoBinder if the message was not created by a client.
//
// Example:
//
//	reply, err := msg.Reply("Got it!", &params.SendMessage{ParseMode: params.ParseModeHTML})
func (m *Message) Reply(text string, opts ...*params.SendMessage) (*Message, error) {
	if m.binder == nil {
		return nil, ErrNoBinder
	}
	return m.binder.BoundSend(m.ChatID, text, m.ID, opts...)
}

// Send sends a text message in the same chat without replying to any message.
// Returns ErrNoBinder if the message was not created by a client.
func (m *Message) Send(text string, opts ...*params.SendMessage) (*Message, error) {
	if m.binder == nil {
		return nil, ErrNoBinder
	}
	return m.binder.BoundSend(m.ChatID, text, 0, opts...)
}

// Forward forwards this message to another chat.
// Returns ErrNoBinder if the message was not created by a client.
//
// Example:
//
//	fwd, err := msg.Forward(targetChatID, &params.ForwardMessages{DropAuthor: true})
//	fmt.Printf("Forwarded as message %d\n", fwd.ID)
func (m *Message) Forward(chatID int64, opts ...*params.ForwardMessages) (*Message, error) {
	if m.binder == nil {
		return nil, ErrNoBinder
	}
	return m.binder.BoundForward(chatID, m.ChatID, m.ID, opts...)
}

// Copy copies this message into another chat without the forward header.
// Returns the new message ID. Returns ErrNoBinder if the message was not created
// by a client.
func (m *Message) Copy(chatID int64, opts ...*params.CopyMessage) (int64, error) {
	if m.binder == nil {
		return 0, ErrNoBinder
	}
	return m.binder.BoundCopy(chatID, m.ChatID, m.ID, opts...)
}

// Edit modifies the text content of this message.
// Only possible for messages sent by the current user.
// Returns ErrNoBinder if the message was not created by a client.
//
// Example:
//
//	edited, err := msg.Edit("<i>Updated text</i>", &params.EditMessage{ParseMode: params.ParseModeHTML})
func (m *Message) Edit(text string, opts ...*params.EditMessage) (*Message, error) {
	if m.binder == nil {
		return nil, ErrNoBinder
	}
	return m.binder.BoundEdit(m.ChatID, m.ID, text, opts...)
}

// EditCaption changes the caption of this media message.
// Returns ErrNoBinder if the message was not created by a client.
func (m *Message) EditCaption(caption string, opts ...*params.EditMessage) (*Message, error) {
	if m.binder == nil {
		return nil, ErrNoBinder
	}
	return m.binder.BoundEditCaption(m.ChatID, m.ID, caption, opts...)
}

// Delete removes this message from the chat.
// Returns the number of messages deleted. Returns ErrNoBinder if the message was
// not created by a client.
//
// Example:
//
//	count, err := msg.Delete(&params.DeleteMessages{Revoke: true})
//	fmt.Printf("Deleted %d message(s)\n", count)
func (m *Message) Delete(opts ...*params.DeleteMessages) (int, error) {
	if m.binder == nil {
		return 0, ErrNoBinder
	}
	return m.binder.BoundDelete(m.ChatID, []int32{m.ID}, opts...)
}

// React adds one or more emoji reactions to this message.
// Returns ErrNoBinder if the message was not created by a client.
//
// Example:
//
//	err := msg.React("👍", "❤️")
func (m *Message) React(emojis ...string) error {
	if m.binder == nil {
		return ErrNoBinder
	}
	var opts []*params.React
	for _, e := range emojis {
		opts = append(opts, &params.React{Emoji: e})
	}
	return m.binder.BoundReact(m.ChatID, m.ID, opts...)
}

// Pin pins this message in the chat.
// Returns ErrNoBinder if the message was not created by a client.
//
// Example:
//
//	err := msg.Pin(&params.PinMessage{Silent: true})
func (m *Message) Pin(opts ...*params.PinMessage) error {
	if m.binder == nil {
		return ErrNoBinder
	}
	return m.binder.BoundPin(m.ChatID, m.ID, opts...)
}

// Unpin removes this message from the pinned list.
// Returns ErrNoBinder if the message was not created by a client.
func (m *Message) Unpin(opts ...*params.PinMessage) error {
	if m.binder == nil {
		return ErrNoBinder
	}
	return m.binder.BoundUnpin(m.ChatID, m.ID, opts...)
}

// Read marks this message (and all before it) as read.
// Returns ErrNoBinder if the message was not created by a client.
func (m *Message) Read() error {
	if m.binder == nil {
		return ErrNoBinder
	}
	return m.binder.BoundRead(m.ChatID, m.ID)
}

// Download fetches the media attached to this message as raw bytes.
// Returns ErrNoBinder if the message was not created by a client.
//
// Example:
//
//	data, err := msg.Download()
//	if err == nil {
//	    os.WriteFile("photo.jpg", data, 0644)
//	}
func (m *Message) Download(opts ...*params.Download) ([]byte, error) {
	if m.binder == nil {
		return nil, ErrNoBinder
	}
	return m.binder.BoundDownload(m.ChatID, m.ID, opts...)
}

// DownloadTo downloads the media attached to this message to a file and returns
// the absolute file path. If fileName is empty, the file is saved in the
// "downloads" directory with an auto-generated name. Paths ending with "/" are
// treated as directories. Non-existent directories are created automatically.
//
// Returns ErrNoBinder if the message was not created by a client.
//
// Example:
//
//	// Auto-generated path in ./downloads/
//	path, err := msg.DownloadTo("")
//
//	// Custom path
//	path, err := msg.DownloadTo("/tmp/photo.jpg")
//
//	// Custom directory (auto-generates filename)
//	path, err := msg.DownloadTo("/tmp/media/")
func (m *Message) DownloadTo(fileName string, progress params.ProgressFunc) (string, error) {
	if m.binder == nil {
		return "", ErrNoBinder
	}
	return m.binder.BoundDownloadTo(m.ChatID, m.ID, fileName, &params.Download{FileName: fileName, Progress: progress})
}

// ReplyMedia sends media to the same chat, quoting this message as a reply.
// Returns ErrNoBinder if the message was not created by a client.
func (m *Message) ReplyMedia(media tg.InputMediaClass, caption string, opts ...*params.SendMessage) (*Message, error) {
	if m.binder == nil {
		return nil, ErrNoBinder
	}
	return m.binder.BoundSendMedia(m.ChatID, media, caption, m.ID, opts...)
}

// SendMedia sends media to the same chat without replying to any message.
// Returns ErrNoBinder if the message was not created by a client.
func (m *Message) SendMedia(media tg.InputMediaClass, caption string, opts ...*params.SendMessage) (*Message, error) {
	if m.binder == nil {
		return nil, ErrNoBinder
	}
	return m.binder.BoundSendMedia(m.ChatID, media, caption, 0, opts...)
}

// EditMedia replaces the media content of this message.
// Returns ErrNoBinder if the message was not created by a client.
func (m *Message) EditMedia(media tg.InputMediaClass) (*Message, error) {
	if m.binder == nil {
		return nil, ErrNoBinder
	}
	return m.binder.BoundEditMedia(m.ChatID, m.ID, media)
}

// EditReplyMarkup changes only the inline keyboard of this message.
// Returns ErrNoBinder if the message was not created by a client.
func (m *Message) EditReplyMarkup(markup tg.ReplyMarkupClass) (*Message, error) {
	if m.binder == nil {
		return nil, ErrNoBinder
	}
	return m.binder.BoundEditReplyMarkup(m.ChatID, m.ID, markup)
}

func (m *Message) ReplyAnimation(file *InputFile, caption string, opts ...*params.SendAnimation) (*Message, error) {
	if m.binder == nil {
		return nil, ErrNoBinder
	}
	return m.binder.BoundReplyAnimation(m.ChatID, file, caption, m.ID, opts...)
}

func (m *Message) ReplyAudio(file *InputFile, caption string, opts ...*params.SendAudio) (*Message, error) {
	if m.binder == nil {
		return nil, ErrNoBinder
	}
	return m.binder.BoundReplyAudio(m.ChatID, file, caption, m.ID, opts...)
}

func (m *Message) ReplyDocument(file *InputFile, caption string, opts ...*params.SendDocument) (*Message, error) {
	if m.binder == nil {
		return nil, ErrNoBinder
	}
	return m.binder.BoundReplyDocument(m.ChatID, file, caption, m.ID, opts...)
}

func (m *Message) ReplyPhoto(file *InputFile, caption string, opts ...*params.SendPhoto) (*Message, error) {
	if m.binder == nil {
		return nil, ErrNoBinder
	}
	return m.binder.BoundReplyPhoto(m.ChatID, file, caption, m.ID, opts...)
}

func (m *Message) ReplyVideo(file *InputFile, caption string, opts ...*params.SendVideo) (*Message, error) {
	if m.binder == nil {
		return nil, ErrNoBinder
	}
	return m.binder.BoundReplyVideo(m.ChatID, file, caption, m.ID, opts...)
}

func (m *Message) ReplyVideoNote(file *InputFile, opts ...*params.SendVideoNote) (*Message, error) {
	if m.binder == nil {
		return nil, ErrNoBinder
	}
	return m.binder.BoundReplyVideoNote(m.ChatID, file, m.ID, opts...)
}

func (m *Message) ReplyVoice(file *InputFile, caption string, opts ...*params.SendVoice) (*Message, error) {
	if m.binder == nil {
		return nil, ErrNoBinder
	}
	return m.binder.BoundReplyVoice(m.ChatID, file, caption, m.ID, opts...)
}

func (m *Message) ReplySticker(file *InputFile, opts ...*params.SendSticker) (*Message, error) {
	if m.binder == nil {
		return nil, ErrNoBinder
	}
	return m.binder.BoundReplySticker(m.ChatID, file, m.ID, opts...)
}

// ReplyText is an alias for Reply, provided for naming consistency with the
// other Reply* helpers.
func (m *Message) ReplyText(text string, opts ...*params.SendMessage) (*Message, error) {
	return m.Reply(text, opts...)
}

// Answer sends a text message in the same chat without replying. Semantically
// equivalent to Send; named for use in callback/query contexts.
func (m *Message) Answer(text string, opts ...*params.SendMessage) (*Message, error) {
	if m.binder == nil {
		return nil, ErrNoBinder
	}
	return m.binder.BoundSend(m.ChatID, text, 0, opts...)
}

func (m *Message) AnswerAnimation(file *InputFile, caption string, opts ...*params.SendAnimation) (*Message, error) {
	if m.binder == nil {
		return nil, ErrNoBinder
	}
	return m.binder.BoundAnswerAnimation(m.ChatID, file, caption, opts...)
}

func (m *Message) AnswerAudio(file *InputFile, caption string, opts ...*params.SendAudio) (*Message, error) {
	if m.binder == nil {
		return nil, ErrNoBinder
	}
	return m.binder.BoundAnswerAudio(m.ChatID, file, caption, opts...)
}

func (m *Message) AnswerDocument(file *InputFile, caption string, opts ...*params.SendDocument) (*Message, error) {
	if m.binder == nil {
		return nil, ErrNoBinder
	}
	return m.binder.BoundAnswerDocument(m.ChatID, file, caption, opts...)
}

func (m *Message) AnswerPhoto(file *InputFile, caption string, opts ...*params.SendPhoto) (*Message, error) {
	if m.binder == nil {
		return nil, ErrNoBinder
	}
	return m.binder.BoundAnswerPhoto(m.ChatID, file, caption, opts...)
}

func (m *Message) AnswerVideo(file *InputFile, caption string, opts ...*params.SendVideo) (*Message, error) {
	if m.binder == nil {
		return nil, ErrNoBinder
	}
	return m.binder.BoundAnswerVideo(m.ChatID, file, caption, opts...)
}

func (m *Message) AnswerVideoNote(file *InputFile, opts ...*params.SendVideoNote) (*Message, error) {
	if m.binder == nil {
		return nil, ErrNoBinder
	}
	return m.binder.BoundAnswerVideoNote(m.ChatID, file, opts...)
}

func (m *Message) AnswerVoice(file *InputFile, caption string, opts ...*params.SendVoice) (*Message, error) {
	if m.binder == nil {
		return nil, ErrNoBinder
	}
	return m.binder.BoundAnswerVoice(m.ChatID, file, caption, opts...)
}

func (m *Message) AnswerSticker(file *InputFile, opts ...*params.SendSticker) (*Message, error) {
	if m.binder == nil {
		return nil, ErrNoBinder
	}
	return m.binder.BoundAnswerSticker(m.ChatID, file, opts...)
}

// AnswerMedia sends media to the same chat without replying to any message.
func (m *Message) AnswerMedia(media tg.InputMediaClass, caption string, opts ...*params.SendMessage) (*Message, error) {
	if m.binder == nil {
		return nil, ErrNoBinder
	}
	return m.binder.BoundSendMedia(m.ChatID, media, caption, 0, opts...)
}

// AnswerMediaGroup sends an album of media items to the same chat without
// replying to any message.
func (m *Message) AnswerMediaGroup(media []tg.InputMediaClass, opts ...*params.SendMediaGroup) ([]*Message, error) {
	if m.binder == nil {
		return nil, ErrNoBinder
	}
	return m.binder.BoundSendMediaGroup(m.ChatID, media, 0, opts...)
}

// ReplyContact sends a contact card as a reply to this message.
func (m *Message) ReplyContact(phone, firstName, lastName string, opts ...*params.SendContact) (*Message, error) {
	if m.binder == nil {
		return nil, ErrNoBinder
	}
	return m.binder.BoundSendContact(m.ChatID, phone, firstName, lastName, m.ID, opts...)
}

// AnswerContact sends a contact card without replying.
func (m *Message) AnswerContact(phone, firstName, lastName string, opts ...*params.SendContact) (*Message, error) {
	if m.binder == nil {
		return nil, ErrNoBinder
	}
	return m.binder.BoundSendContact(m.ChatID, phone, firstName, lastName, 0, opts...)
}

// ReplyLocation sends a geographic location as a reply to this message.
func (m *Message) ReplyLocation(lat, lng float64, opts ...*params.SendLocation) (*Message, error) {
	if m.binder == nil {
		return nil, ErrNoBinder
	}
	return m.binder.BoundSendLocation(m.ChatID, lat, lng, m.ID, opts...)
}

// AnswerLocation sends a geographic location without replying.
func (m *Message) AnswerLocation(lat, lng float64, opts ...*params.SendLocation) (*Message, error) {
	if m.binder == nil {
		return nil, ErrNoBinder
	}
	return m.binder.BoundSendLocation(m.ChatID, lat, lng, 0, opts...)
}

// ReplyVenue sends a named venue as a reply to this message.
func (m *Message) ReplyVenue(lat, lng float64, title, address string, opts ...*params.SendVenue) (*Message, error) {
	if m.binder == nil {
		return nil, ErrNoBinder
	}
	return m.binder.BoundSendVenue(m.ChatID, lat, lng, title, address, m.ID, opts...)
}

// AnswerVenue sends a named venue without replying.
func (m *Message) AnswerVenue(lat, lng float64, title, address string, opts ...*params.SendVenue) (*Message, error) {
	if m.binder == nil {
		return nil, ErrNoBinder
	}
	return m.binder.BoundSendVenue(m.ChatID, lat, lng, title, address, 0, opts...)
}

// ReplyPoll sends a poll as a reply to this message.
func (m *Message) ReplyPoll(question string, options []string, opts ...*params.SendPoll) (*Message, error) {
	if m.binder == nil {
		return nil, ErrNoBinder
	}
	return m.binder.BoundSendPoll(m.ChatID, question, options, m.ID, opts...)
}

// AnswerPoll sends a poll without replying.
func (m *Message) AnswerPoll(question string, options []string, opts ...*params.SendPoll) (*Message, error) {
	if m.binder == nil {
		return nil, ErrNoBinder
	}
	return m.binder.BoundSendPoll(m.ChatID, question, options, 0, opts...)
}

// ReplyDice sends a dice roll as a reply to this message.
func (m *Message) ReplyDice(emoji string, opts ...*params.SendDice) (*Message, error) {
	if m.binder == nil {
		return nil, ErrNoBinder
	}
	return m.binder.BoundSendDice(m.ChatID, emoji, m.ID, opts...)
}

// AnswerDice sends a dice roll without replying.
func (m *Message) AnswerDice(emoji string, opts ...*params.SendDice) (*Message, error) {
	if m.binder == nil {
		return nil, ErrNoBinder
	}
	return m.binder.BoundSendDice(m.ChatID, emoji, 0, opts...)
}

// ReplyGame sends a game as a reply to this message.
func (m *Message) ReplyGame(gameShortName string, opts ...*params.SendGame) (*Message, error) {
	if m.binder == nil {
		return nil, ErrNoBinder
	}
	return m.binder.BoundSendGame(m.ChatID, gameShortName, m.ID, opts...)
}

// AnswerGame sends a game without replying.
func (m *Message) AnswerGame(gameShortName string, opts ...*params.SendGame) (*Message, error) {
	if m.binder == nil {
		return nil, ErrNoBinder
	}
	return m.binder.BoundSendGame(m.ChatID, gameShortName, 0, opts...)
}

func (m *Message) ReplyCachedMedia(file *InputFile, caption string, opts ...*params.SendDocument) (*Message, error) {
	if m.binder == nil {
		return nil, ErrNoBinder
	}
	return m.binder.BoundReplyDocument(m.ChatID, file, caption, m.ID, opts...)
}

func (m *Message) AnswerCachedMedia(file *InputFile, caption string, opts ...*params.SendDocument) (*Message, error) {
	if m.binder == nil {
		return nil, ErrNoBinder
	}
	return m.binder.BoundAnswerDocument(m.ChatID, file, caption, opts...)
}

// ReplyMediaGroup sends an album of media items as a reply to this message.
func (m *Message) ReplyMediaGroup(media []tg.InputMediaClass, opts ...*params.SendMediaGroup) ([]*Message, error) {
	if m.binder == nil {
		return nil, ErrNoBinder
	}
	return m.binder.BoundSendMediaGroup(m.ChatID, media, m.ID, opts...)
}

// ReplyChatAction sends a chat action indicator (e.g. typing) to the same chat.
func (m *Message) ReplyChatAction(action tg.SendMessageActionClass) error {
	if m.binder == nil {
		return ErrNoBinder
	}
	return m.binder.BoundSendChatAction(m.ChatID, action)
}

// ReplyInlineBotResult sends an inline bot result as a reply to this message.
func (m *Message) ReplyInlineBotResult(queryID int64, resultID string, opts ...*params.SendInlineBotResult) (*Message, error) {
	if m.binder == nil {
		return nil, ErrNoBinder
	}
	return m.binder.BoundSendInlineBotResult(m.ChatID, queryID, resultID, m.ID, opts...)
}

// AnswerInlineBotResult sends an inline bot result without replying.
func (m *Message) AnswerInlineBotResult(queryID int64, resultID string, opts ...*params.SendInlineBotResult) (*Message, error) {
	if m.binder == nil {
		return nil, ErrNoBinder
	}
	return m.binder.BoundSendInlineBotResult(m.ChatID, queryID, resultID, 0, opts...)
}

// EditText is an alias for Edit, provided for naming consistency.
func (m *Message) EditText(text string, opts ...*params.EditMessage) (*Message, error) {
	return m.Edit(text, opts...)
}

// EditLiveLocation updates the location of a live location message. Not yet
// implemented.
func (m *Message) EditLiveLocation(lat, lng float64) (*Message, error) {
	return nil, m.binder.BoundStub("EditLiveLocation")
}

// StopLiveLocation stops a live location sharing. Not yet implemented.
func (m *Message) StopLiveLocation() (*Message, error) {
	return nil, m.binder.BoundStub("StopLiveLocation")
}

// CopyMediaGroup copies all messages in this message's album into another chat.
func (m *Message) CopyMediaGroup(chatID int64) ([]*Message, error) {
	if m.binder == nil {
		return nil, ErrNoBinder
	}
	return m.binder.BoundCopyMediaGroup(chatID, m.ChatID, m.ID)
}

// Vote casts a vote in a poll message using the selected option byte sequences.
func (m *Message) Vote(options [][]byte) error {
	if m.binder == nil {
		return ErrNoBinder
	}
	return m.binder.BoundVote(m.ChatID, m.ID, options)
}

// RetractVote withdraws the current user's vote in a poll.
func (m *Message) RetractVote() error {
	if m.binder == nil {
		return ErrNoBinder
	}
	return m.binder.BoundRetractVote(m.ChatID, m.ID)
}

// GetMediaGroup retrieves all messages that belong to the same album as this
// message.
func (m *Message) GetMediaGroup() ([]*Message, error) {
	if m.binder == nil {
		return nil, ErrNoBinder
	}
	return m.binder.BoundGetMediaGroup(m.ChatID, m.ID)
}

// View is an alias for Read; marks this message as read.
func (m *Message) View() error {
	return m.Read()
}

// Click simulates a button press on this message. Not yet implemented.
func (m *Message) Click(buttonIndex int) error {
	if m.binder == nil {
		return ErrNoBinder
	}
	return m.binder.BoundStub("Click")
}

// Pay initiates a payment for an invoice message. Not yet implemented.
func (m *Message) Pay() error {
	if m.binder == nil {
		return ErrNoBinder
	}
	return m.binder.BoundStub("Pay")
}

// ReplyPaidMedia sends paid media as a reply. Not yet implemented.
func (m *Message) ReplyPaidMedia(caption string, opts ...*params.SendMessage) (*Message, error) {
	if m.binder == nil {
		return nil, ErrNoBinder
	}
	return nil, m.binder.BoundStub("ReplyPaidMedia")
}

// AnswerPaidMedia sends paid media without replying. Not yet implemented.
func (m *Message) AnswerPaidMedia(caption string, opts ...*params.SendMessage) (*Message, error) {
	if m.binder == nil {
		return nil, ErrNoBinder
	}
	return nil, m.binder.BoundStub("AnswerPaidMedia")
}

// ReplyInvoice sends an invoice as a reply. Not yet implemented.
func (m *Message) ReplyInvoice(title, description string, opts ...*params.SendMessage) (*Message, error) {
	if m.binder == nil {
		return nil, ErrNoBinder
	}
	return nil, m.binder.BoundStub("ReplyInvoice")
}

// AnswerInvoice sends an invoice without replying. Not yet implemented.
func (m *Message) AnswerInvoice(title, description string, opts ...*params.SendMessage) (*Message, error) {
	if m.binder == nil {
		return nil, ErrNoBinder
	}
	return nil, m.binder.BoundStub("AnswerInvoice")
}

func (m *Message) ReplyChecklist(checklist *tg.InputMediaTodo, opts ...*params.SendChecklist) (*Message, error) {
	if m.binder == nil {
		return nil, ErrNoBinder
	}
	return m.binder.BoundReplyChecklist(m.ChatID, checklist, m.ID, opts...)
}

func (m *Message) AnswerChecklist(checklist *tg.InputMediaTodo, opts ...*params.SendChecklist) (*Message, error) {
	if m.binder == nil {
		return nil, ErrNoBinder
	}
	return m.binder.BoundReplyChecklist(m.ChatID, checklist, 0, opts...)
}

func (m *Message) EditChecklist(checklist *tg.InputMediaTodo, opts ...*params.EditMessage) (*Message, error) {
	if m.binder == nil {
		return nil, ErrNoBinder
	}
	media := tg.InputMediaClass(checklist)
	return m.binder.BoundEditMedia(m.ChatID, m.ID, media)
}

// AcceptGiftPurchaseOffer accepts a gift purchase offer. Not yet implemented.
func (m *Message) AcceptGiftPurchaseOffer() error {
	if m.binder == nil {
		return ErrNoBinder
	}
	return m.binder.BoundStub("AcceptGiftPurchaseOffer")
}

// RejectGiftPurchaseOffer rejects a gift purchase offer. Not yet implemented.
func (m *Message) RejectGiftPurchaseOffer() error {
	if m.binder == nil {
		return ErrNoBinder
	}
	return m.binder.BoundStub("RejectGiftPurchaseOffer")
}

// Summarize generates a summary of the message content. Not yet implemented.
func (m *Message) Summarize() (string, error) {
	if m.binder == nil {
		return "", ErrNoBinder
	}
	return "", m.binder.BoundStub("Summarize")
}
