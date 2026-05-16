package types

import (
	"fmt"

	"github.com/mtgo-labs/mtgo/tg"
)

// InlineKeyboardButton represents a button within an inline keyboard attached
// to a message. Inline keyboards remain attached to the message and trigger
// callbacks or URL navigation when pressed.
//
// Example:
//
//	btn := types.InlineKeyboardButton{
//	    Text:         "Visit",
//	    URL:          "https://example.com",
//	}
//	_ = btn
type InlineKeyboardButton struct {
	Text                         string
	URL                          string
	CallbackData                 []byte
	SwitchInlineQuery            string
	SwitchInlineQueryCurrentChat string
	LoginURL                     *LoginURL
	WebApp                       *WebAppInfo
	CallbackGame                 *CallbackGame
	UserID                       int64
	IsRequiresPassword           bool
	Pay                          bool
	CopyText                     string
	IconCustomEmojiID            string
	Style                        ButtonStyle
}

// LoginURL describes a direct URL-based login flow triggered from an inline
// button. Used for seamlessly authenticating users on external websites via
// Telegram.
type LoginURL struct {
	// URL is the target URL the user is redirected to after successful login.
	URL string
	// Domain is the domain of the website the user is logging into.
	Domain string
	// ForwardText is the new text of the button in forwarded messages.
	ForwardText string
	// BotUsername is the username of a bot used for user authorization.
	BotUsername string
	// RequestWriteAccess indicates the bot requests permission to send messages to the user.
	RequestWriteAccess bool
	// ButtonID is the identifier of the button.
	ButtonID int
}

// WebAppInfo holds the URL of a Telegram Web App opened by a keyboard button.
// Web Apps run inside the Telegram client and can interact with the bot API.
type WebAppInfo struct {
	// URL is the HTTPS URL of the Web App to open.
	URL string
}

// KeyboardButton represents a button in a reply keyboard that can request user
// input such as location, contact, or chat selection. Reply keyboards replace
// the user's default keyboard and are shown below the text input field.
//
// Example:
//
//	btn := types.KeyboardButton{
//	    Text:           "Share my phone",
//	    RequestContact: true,
//	}
//	_ = btn
type KeyboardButton struct {
	Text              string
	RequestChat       *KeyboardButtonRequestChat
	RequestUsers      *KeyboardButtonRequestUsers
	RequestPoll       *KeyboardButtonPollType
	RequestManagedBot *KeyboardButtonRequestManagedBot
	WebApp            *WebAppInfo
	RequestLocation   bool
	RequestContact    bool
	IconCustomEmojiID string
	Style             ButtonStyle
}

// KeyboardButtonRequestChat carries parameters for a button that asks the user
// to select a chat or channel. Returned via a ChatShared service message.
type KeyboardButtonRequestChat struct {
	RequestID               int64
	ChatIsChannel           bool
	IsChatForum             bool
	IsChatHasUsername       bool
	IsChatCreated           bool
	IsBotMember             bool
	ButtonID                int
	BotChatRights           *ChatAdminRights
	UserAdministratorRights *ChatAdministratorRights
	BotAdministratorRights  *ChatAdministratorRights
	IsRequestTitle          bool
	IsRequestUsername       bool
	IsRequestPhoto          bool
	MaxQuantity             int
}

// KeyboardButtonRequestUsers carries parameters for a button that asks the user
// to select one or more users. Returned via a UsersShared service message.
type KeyboardButtonRequestUsers struct {
	RequestID         int64
	Max               int32
	ButtonID          int
	IsUserBot         bool
	IsUserPremium     bool
	MaxQuantity       int
	IsRequestName     bool
	IsRequestUsername bool
	IsRequestPhoto    bool
}

// KeyboardButtonPollType qualifies what kind of poll the button should request.
// An empty Type allows any poll type.
type KeyboardButtonPollType struct {
	// Type restricts the poll to quiz or regular mode. Leave zero-valued to allow both.
	Type PollType
}

// ReplyKeyboardRemove instructs the client to hide the bot's reply keyboard.
// Sent as part of a reply markup to clean up the keyboard after use.
type ReplyKeyboardRemove struct {
	// Selective indicates the keyboard is removed only for users mentioned in
	// the message text or for the sender of the replied-to message.
	Selective bool
}

// ForceReply forces the client to display a reply interface to the user, even
// if the bot could receive messages without it. Useful for one-off prompts.
type ForceReply struct {
	// Selective indicates the reply interface is shown only for users mentioned
	// in the message text or for the sender of the replied-to message.
	Selective bool
	// Placeholder is the hint text shown in the reply input field, e.g. "Enter your name…".
	Placeholder string
}

// GameHighScore represents a single entry in a game's high-score table. Used to
// display rankings across players for a Telegram game.
type GameHighScore struct {
	User     *User
	UserID   int64
	Score    int32
	Position int32
}

// SentWebAppMessage is returned when a Web App sends a message via
// answerWebAppQuery. Contains the identifier of the sent inline message.
type SentWebAppMessage struct {
	// InlineMessageID is the identifier of the inline message that was sent
	// from the Web App, formatted as "dc:id:access_hash".
	InlineMessageID string
}

// ParseGameHighScore converts an MTProto HighScore to a GameHighScore.
// Returns nil if raw is nil.
func ParseGameHighScore(raw *tg.HighScore, users map[int64]tg.UserClass) *GameHighScore {
	if raw == nil {
		return nil
	}
	hs := &GameHighScore{
		Position: raw.Pos,
		UserID:   raw.UserID,
		Score:    raw.Score,
	}
	if users != nil {
		hs.User = getUser(users, raw.UserID)
	}
	return hs
}

// ParseSentWebAppMessage converts an MTProto WebViewMessageSent to a
// SentWebAppMessage. Returns nil if raw is nil.
func ParseSentWebAppMessage(raw *tg.WebViewMessageSent) *SentWebAppMessage {
	if raw == nil {
		return nil
	}
	out := &SentWebAppMessage{}
	if raw.MsgID != nil {
		out.InlineMessageID = formatInlineMessageID(raw.MsgID)
	}
	return out
}

func formatInlineMessageID(raw tg.InputBotInlineMessageIDClass) string {
	switch v := raw.(type) {
	case *tg.InputBotInlineMessageID:
		return fmt.Sprintf("%d:%d:%d", v.DCID, v.ID, v.AccessHash)
	case *tg.InputBotInlineMessageID64:
		return fmt.Sprintf("%d:%d:%d:%d", v.DCID, v.OwnerID, v.ID, v.AccessHash)
	default:
		return ""
	}
}
