package types

// MessageEntityType represents the type of a text entity in a message,
// such as a mention, hashtag, URL, or formatting style.
type MessageEntityType string

const (
	// MessageEntityTypeMention is a @username reference in the text.
	MessageEntityTypeMention MessageEntityType = "mention"
	// MessageEntityTypeHashtag is a #hashtag in the text.
	MessageEntityTypeHashtag MessageEntityType = "hashtag"
	// MessageEntityTypeCashtag is a $cashtag (e.g. ticker symbol) in the text.
	MessageEntityTypeCashtag MessageEntityType = "cashtag"
	// MessageEntityTypeBotCommand is a /command directed at a bot.
	MessageEntityTypeBotCommand MessageEntityType = "bot_command"
	// MessageEntityTypeURL is a bare HTTP/HTTPS URL in the text.
	MessageEntityTypeURL MessageEntityType = "url"
	// MessageEntityTypeEmail is an email address in the text.
	MessageEntityTypeEmail MessageEntityType = "email"
	// MessageEntityTypePhoneNumber is a phone number in the text.
	MessageEntityTypePhoneNumber MessageEntityType = "phone_number"
	// MessageEntityTypeBold indicates bold formatting.
	MessageEntityTypeBold MessageEntityType = "bold"
	// MessageEntityTypeItalic indicates italic formatting.
	MessageEntityTypeItalic MessageEntityType = "italic"
	// MessageEntityTypeUnderline indicates underlined text.
	MessageEntityTypeUnderline MessageEntityType = "underline"
	// MessageEntityTypeStrikethrough indicates strikethrough formatting.
	MessageEntityTypeStrikethrough MessageEntityType = "strikethrough"
	// MessageEntityTypeSpoiler indicates spoiler-hidden text revealed on tap.
	MessageEntityTypeSpoiler MessageEntityType = "spoiler"
	// MessageEntityTypeCode indicates monospace inline code.
	MessageEntityTypeCode MessageEntityType = "code"
	// MessageEntityTypePre indicates a pre-formatted code block, with an optional language.
	MessageEntityTypePre MessageEntityType = "pre"
	// MessageEntityTypeBlockquote indicates a block quotation.
	MessageEntityTypeBlockquote MessageEntityType = "blockquote"
	// MessageEntityTypeTextLink is a clickable text URL (different from bare URL entity).
	MessageEntityTypeTextLink MessageEntityType = "text_link"
	// MessageEntityTypeTextMention is a mention of a user without a username, identified by UserID.
	MessageEntityTypeTextMention MessageEntityType = "text_mention"
	// MessageEntityTypeBankCard is a detected bank card number in the text.
	MessageEntityTypeBankCard MessageEntityType = "bank_card"
	// MessageEntityTypeCustomEmoji is a custom emoji rendered from a sticker set.
	MessageEntityTypeCustomEmoji MessageEntityType = "custom_emoji"
	// MessageEntityTypeDateTime is a formatted date/time reference in the text.
	MessageEntityTypeDateTime MessageEntityType = "date_time"
	// MessageEntityTypeUnknown represents an entity type not recognized by this library.
	MessageEntityTypeUnknown MessageEntityType = "unknown"
)

// String returns the string representation of the MessageEntityType.
func (m MessageEntityType) String() string { return string(m) }
