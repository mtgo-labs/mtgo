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
	// Text is the visible label shown to the user on the button.
	Text string
	// URL is the HTTP or tg:// URL opened when the button is pressed.
	URL string
	// CallbackData is the opaque payload delivered to the bot when the user
	// presses a callback button. Use this for stateful button interactions.
	CallbackData []byte
	// SwitchInline asks the client to open an inline query in the current chat
	// or a selected chat, pre-filled with this value.
	SwitchInline string
	// SamePeer indicates that the inline query triggered by SwitchInline should
	// remain in the same peer context rather than prompting the user to pick a chat.
	SamePeer bool
	// LoginURL describes a URL-based Telegram login flow triggered by this button,
	// used for third-party account linking.
	LoginURL *LoginURL
	// WebApp holds the URL of a Telegram Web App launched when the button is pressed.
	WebApp *WebAppInfo
	// Game indicates that this button launches a Telegram game when pressed.
	Game bool
	// Pay indicates that this button triggers a payment flow when pressed.
	Pay bool
}

// LoginURL describes a direct URL-based login flow triggered from an inline
// button. Used for seamlessly authenticating users on external websites via
// Telegram.
type LoginURL struct {
	// URL is the target URL the user is redirected to after successful login.
	URL string
	// Domain is the domain of the website the user is logging into.
	Domain string
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
	// Text is the visible label shown to the user on the button.
	Text string
	// RequestChat carries parameters for a button that asks the user to select
	// a chat or channel to share with the bot.
	RequestChat *KeyboardButtonRequestChat
	// RequestUsers carries parameters for a button that asks the user to select
	// one or more users to share with the bot.
	RequestUsers *KeyboardButtonRequestUsers
	// RequestPoll qualifies what kind of poll the button should request the user
	// to create (quiz or regular).
	RequestPoll *KeyboardButtonPollType
	// WebApp holds the URL of a Telegram Web App launched when the button is pressed.
	WebApp *WebAppInfo
	// RequestLocation indicates that the button requests the user to share their
	// current location when pressed.
	RequestLocation bool
	// RequestContact indicates that the button requests the user to share their
	// phone number when pressed.
	RequestContact bool
}

// KeyboardButtonRequestChat carries parameters for a button that asks the user
// to select a chat or channel. Returned via a ChatShared service message.
type KeyboardButtonRequestChat struct {
	// RequestID is a unique identifier for this request, used to correlate the
	// resulting ChatShared service message with this button press.
	RequestID int64
	// ChatIsChannel indicates that only channels should be selectable, not groups.
	ChatIsChannel bool
	// BotChatRights specifies the admin rights the bot requests in the selected chat,
	// if any.
	BotChatRights *ChatAdminRights
}

// KeyboardButtonRequestUsers carries parameters for a button that asks the user
// to select one or more users. Returned via a UsersShared service message.
type KeyboardButtonRequestUsers struct {
	// RequestID is a unique identifier for this request, used to correlate the
	// resulting UsersShared service message with this button press.
	RequestID int64
	// Max is the maximum number of users that can be selected simultaneously.
	Max int32
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
	// UserID is the Telegram user ID of the player.
	UserID int64
	// Score is the player's numeric score in the game.
	Score int32
	// Position is the player's rank in the global high-score table (1-based).
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
func ParseGameHighScore(raw *tg.HighScore) *GameHighScore {
	if raw == nil {
		return nil
	}
	return &GameHighScore{
		Position: raw.Pos,
		UserID:   raw.UserID,
		Score:    raw.Score,
	}
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
