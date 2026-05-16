package params

import (
	tl "github.com/mtgo-labs/mtgo/tg"
)

// InlineKeyboard builds an inline keyboard markup from the provided rows of
// buttons. Each row is a slice of KeyboardButton values.
//
// Example:
//
//	markup := params.InlineKeyboard(
//	    []params.KeyboardButton{params.ButtonCB("Yes", "yes"), params.ButtonCB("No", "no")},
//	)
func InlineKeyboard(rows ...[]KeyboardButton) tl.ReplyMarkupClass {
	tlRows := make([]*tl.KeyboardButtonRow, len(rows))
	for i, row := range rows {
		buttons := make([]tl.KeyboardButtonClass, len(row))
		for j, b := range row {
			buttons[j] = b.toInlineTL()
		}
		tlRows[i] = &tl.KeyboardButtonRow{Buttons: buttons}
	}
	return &tl.ReplyInlineMarkup{Rows: tlRows}
}

// ReplyKeyboard builds a reply keyboard markup from the provided rows of
// buttons. Each row is a slice of KeyboardButton values with text only.
//
// Example:
//
//	markup := params.ReplyKeyboard(
//	    []params.KeyboardButton{params.Button("Option A"), params.Button("Option B")},
//	    []params.KeyboardButton{params.Button("Cancel")},
//	)
func ReplyKeyboard(rows ...[]KeyboardButton) tl.ReplyMarkupClass {
	tlRows := make([]*tl.KeyboardButtonRow, len(rows))
	for i, row := range rows {
		buttons := make([]tl.KeyboardButtonClass, len(row))
		for j, b := range row {
			buttons[j] = &tl.KeyboardButton{Text: b.Text}
		}
		tlRows[i] = &tl.KeyboardButtonRow{Buttons: buttons}
	}
	return &tl.ReplyKeyboardMarkup{Rows: tlRows}
}

// RemoveKeyboard returns a reply markup that removes the current custom
// keyboard from the chat.
//
// Example:
//
//	markup := params.RemoveKeyboard()
func RemoveKeyboard() tl.ReplyMarkupClass {
	return &tl.ReplyKeyboardHide{}
}

// ForceReplyKeyboard returns a reply markup that forces the user to reply to
// the message.
//
// Example:
//
//	markup := params.ForceReplyKeyboard()
func ForceReplyKeyboard() tl.ReplyMarkupClass {
	return &tl.ReplyKeyboardForceReply{}
}

// KeyboardButton represents a button used in inline and reply keyboards. Set
// the appropriate field to control the button type (plain text, URL, callback,
// switch inline, game, or payment).
//
// Example:
//
//	btn := params.KeyboardButton{Text: "Open", URL: "https://example.com"}
type KeyboardButton struct {
	Text   string
	URL    string
	Data   []byte
	Switch string
	Game   bool
	Pay    bool
}

func (b KeyboardButton) toInlineTL() tl.KeyboardButtonClass {
	if b.URL != "" {
		return &tl.KeyboardButtonURL{Text: b.Text, URL: b.URL}
	}
	if len(b.Data) > 0 {
		return &tl.KeyboardButtonCallback{Text: b.Text, Data: b.Data}
	}
	if b.Switch != "" {
		return &tl.KeyboardButtonSwitchInline{Text: b.Text, Query: b.Switch}
	}
	if b.Game {
		return &tl.KeyboardButtonGame{Text: b.Text}
	}
	if b.Pay {
		return &tl.KeyboardButtonBuy{Text: b.Text}
	}
	return &tl.KeyboardButton{Text: b.Text}
}

// Button creates a plain text keyboard button.
//
// Example:
//
//	btn := params.Button("Click me")
func Button(text string) KeyboardButton {
	return KeyboardButton{Text: text}
}

// ButtonURL creates a keyboard button that opens the given URL when pressed.
//
// Example:
//
//	btn := params.ButtonURL("Visit site", "https://example.com")
func ButtonURL(text, url string) KeyboardButton {
	return KeyboardButton{Text: text, URL: url}
}

// ButtonCB creates a callback keyboard button that sends the data payload back
// to the bot when pressed.
//
// Example:
//
//	btn := params.ButtonCB("Approve", "approve:42")
func ButtonCB(text, data string) KeyboardButton {
	return KeyboardButton{Text: text, Data: []byte(data)}
}

// ButtonSwitch creates a keyboard button that prompts the user to select a chat
// and inserts the query into the input field.
//
// Example:
//
//	btn := params.ButtonSwitch("Share", "search query")
func ButtonSwitch(text, query string) KeyboardButton {
	return KeyboardButton{Text: text, Switch: query}
}

// ButtonGame creates a keyboard button that launches a game.
//
// Example:
//
//	btn := params.ButtonGame("Play Now")
func ButtonGame(text string) KeyboardButton {
	return KeyboardButton{Text: text, Game: true}
}

// ButtonPay creates a keyboard button that initiates a payment.
//
// Example:
//
//	btn := params.ButtonPay("Pay $5")
func ButtonPay(text string) KeyboardButton {
	return KeyboardButton{Text: text, Pay: true}
}

// ReplyToMsg creates a reference that replies to the message with the given ID.
//
// Example:
//
//	ref := params.ReplyToMsg(123)
func ReplyToMsg(msgID int32) tl.InputReplyToClass {
	return &tl.InputReplyToMessage{ReplyToMsgID: msgID}
}

// ReplyToMsgQuote creates a reference that replies to the message with the
// given ID and includes a quoted excerpt starting at offset.
//
// Example:
//
//	ref := params.ReplyToMsgQuote(123, "hello world", 0)
func ReplyToMsgQuote(msgID int32, quote string, offset int32) tl.InputReplyToClass {
	var flags tl.Fields
	flags.Set(0)
	return &tl.InputReplyToMessage{
		Flags:        flags,
		ReplyToMsgID: msgID,
		QuoteText:    quote,
		QuoteOffset:  offset,
	}
}

// ReplyToStory creates a reference that replies to the story with the given ID.
//
// Example:
//
//	ref := params.ReplyToStory(456)
func ReplyToStory(storyID int32) tl.InputReplyToClass {
	return &tl.InputReplyToStory{StoryID: storyID}
}

// PeerUser creates an InputPeerUser reference for the given user ID.
//
// Example:
//
//	peer := params.PeerUser(987654321)
func PeerUser(id int64) tl.InputPeerClass {
	return &tl.InputPeerUser{UserID: id}
}

// PeerChat creates an InputPeerChat reference for the given chat ID.
//
// Example:
//
//	peer := params.PeerChat(123)
func PeerChat(id int64) tl.InputPeerClass {
	return &tl.InputPeerChat{ChatID: id}
}

// PeerChannel creates an InputPeerChannel reference for the given channel ID.
//
// Example:
//
//	peer := params.PeerChannel(456)
func PeerChannel(id int64) tl.InputPeerClass {
	return &tl.InputPeerChannel{ChannelID: id}
}

// PeerFromID auto-detects the peer type from a Telegram ID: positive IDs are
// users, IDs in the -1000000000 range are channels, and other negative IDs are
// chats.
//
// Example:
//
//	user := params.PeerFromID(987654321)    // *tl.InputPeerUser
//	chat := params.PeerFromID(-123)         // *tl.InputPeerChat
//	channel := params.PeerFromID(-1000000123) // *tl.InputPeerChannel
func PeerFromID(id int64) tl.InputPeerClass {
	if id > 0 {
		return &tl.InputPeerUser{UserID: id}
	}
	if id <= -1000000000 {
		return &tl.InputPeerChannel{ChannelID: -1000000000 - id}
	}
	return &tl.InputPeerChat{ChatID: -id}
}

// EntityType enumerates the supported message entity formatting types.
type EntityType int

const (
	EntityMention EntityType = iota
	EntityHashtag
	EntityBotCommand
	EntityURL
	EntityEmail
	EntityBold
	EntityItalic
	EntityCode
	EntityPre
	EntityUnderline
	EntityStrikethrough
	EntitySpoiler
	EntityBlockquote
	EntityCustomEmoji
	EntityExpandableBlockquote
)

// Entity represents a single formatting entity in a message, such as bold,
// italic, or code. Convert to TL form with ToTL.
//
// Example:
//
//	ent := params.Entity{Type: params.EntityBold, Offset: 0, Length: 5}
//	tlEntity := ent.ToTL()
type Entity struct {
	Type          EntityType
	Offset        int32
	Length        int32
	Language      string
	CustomEmojiID int64
}

func (e Entity) ToTL() tl.MessageEntityClass {
	o, l := e.Offset, e.Length
	switch e.Type {
	case EntityMention:
		return &tl.MessageEntityMention{Offset: o, Length: l}
	case EntityHashtag:
		return &tl.MessageEntityHashtag{Offset: o, Length: l}
	case EntityBotCommand:
		return &tl.MessageEntityBotCommand{Offset: o, Length: l}
	case EntityURL:
		return &tl.MessageEntityURL{Offset: o, Length: l}
	case EntityEmail:
		return &tl.MessageEntityEmail{Offset: o, Length: l}
	case EntityBold:
		return &tl.MessageEntityBold{Offset: o, Length: l}
	case EntityItalic:
		return &tl.MessageEntityItalic{Offset: o, Length: l}
	case EntityCode:
		return &tl.MessageEntityCode{Offset: o, Length: l}
	case EntityPre:
		return &tl.MessageEntityPre{Offset: o, Length: l, Language: e.Language}
	case EntityUnderline:
		return &tl.MessageEntityUnderline{Offset: o, Length: l}
	case EntityStrikethrough:
		return &tl.MessageEntityStrike{Offset: o, Length: l}
	case EntitySpoiler:
		return &tl.MessageEntitySpoiler{Offset: o, Length: l}
	case EntityBlockquote:
		return &tl.MessageEntityBlockquote{Offset: o, Length: l}
	case EntityCustomEmoji:
		return &tl.MessageEntityCustomEmoji{Offset: o, Length: l, DocumentID: e.CustomEmojiID}
	case EntityExpandableBlockquote:
		return &tl.MessageEntityBlockquote{Offset: o, Length: l, Collapsed: true}
	}
	return nil
}

// Entities converts a variadic list of Entity values into a slice of TL
// MessageEntityClass values, skipping any nil results.
//
// Example:
//
//	ents := params.Entities(
//	    params.Bold(0, 5),
//	    params.Italic(6, 3),
//	)
func Entities(entities ...Entity) []tl.MessageEntityClass {
	result := make([]tl.MessageEntityClass, 0, len(entities))
	for _, e := range entities {
		if v := e.ToTL(); v != nil {
			result = append(result, v)
		}
	}
	return result
}

// Bold creates a bold formatting entity covering offset through offset+length.
//
// Example:
//
//	ent := params.Bold(0, 5)
func Bold(offset, length int32) Entity {
	return Entity{Type: EntityBold, Offset: offset, Length: length}
}

// Italic creates an italic formatting entity covering offset through
// offset+length.
//
// Example:
//
//	ent := params.Italic(0, 5)
func Italic(offset, length int32) Entity {
	return Entity{Type: EntityItalic, Offset: offset, Length: length}
}

// Code creates a monospace code formatting entity covering offset through
// offset+length.
//
// Example:
//
//	ent := params.Code(0, 7)
func Code(offset, length int32) Entity {
	return Entity{Type: EntityCode, Offset: offset, Length: length}
}

// Pre creates a preformatted code block entity with an optional language
// identifier.
//
// Example:
//
//	ent := params.Pre(0, 20, "go")
func Pre(offset, length int32, lang string) Entity {
	return Entity{Type: EntityPre, Offset: offset, Length: length, Language: lang}
}

// Underline creates an underlined formatting entity covering offset through
// offset+length.
//
// Example:
//
//	ent := params.Underline(0, 5)
func Underline(offset, length int32) Entity {
	return Entity{Type: EntityUnderline, Offset: offset, Length: length}
}

// Strikethrough creates a strikethrough formatting entity covering offset
// through offset+length.
//
// Example:
//
//	ent := params.Strikethrough(0, 5)
func Strikethrough(offset, length int32) Entity {
	return Entity{Type: EntityStrikethrough, Offset: offset, Length: length}
}

// Spoiler creates a spoiler formatting entity covering offset through
// offset+length.
//
// Example:
//
//	ent := params.Spoiler(0, 10)
func Spoiler(offset, length int32) Entity {
	return Entity{Type: EntitySpoiler, Offset: offset, Length: length}
}

// Blockquote creates a blockquote formatting entity covering offset through
// offset+length.
//
// Example:
//
//	ent := params.Blockquote(0, 50)
func Blockquote(offset, length int32) Entity {
	return Entity{Type: EntityBlockquote, Offset: offset, Length: length}
}

// CustomEmoji creates a custom emoji entity using the given document ID.
//
// Example:
//
//	ent := params.CustomEmoji(0, 2, 1234567890)
func CustomEmoji(offset, length int32, documentID int64) Entity {
	return Entity{Type: EntityCustomEmoji, Offset: offset, Length: length, CustomEmojiID: documentID}
}

// Mention creates a @mention formatting entity covering offset through
// offset+length.
//
// Example:
//
//	ent := params.Mention(0, 8)
func Mention(offset, length int32) Entity {
	return Entity{Type: EntityMention, Offset: offset, Length: length}
}
