package types

import (
	"strconv"

	"github.com/mtgo-labs/mtgo/tg"
)

// MessageEntity represents a formatting or semantic entity within a message text,
// such as bold, italic, mention, URL, or code blocks. Each entity is defined by
// its type, UTF-16 offset, and UTF-16 length within the message text.
type MessageEntity struct {
	// Type identifies the kind of entity (bold, italic, mention, URL, etc.).
	Type MessageEntityType
	// Offset is the zero-based UTF-16 code unit offset where the entity starts.
	Offset int
	// Length is the number of UTF-16 code units the entity spans.
	Length int
	// URL is the target URL for text_link entities, or empty for other types.
	URL string
	// User is the mentioned user for text_mention entities, when available.
	User *User
	// UserID is the Telegram user ID for text_mention entities that reference a user
	// without a username, or zero for other types.
	UserID int64
	// Language is the programming language for pre entities, or empty for other types.
	Language string
	// CustomEmojiID is the custom emoji document ID for custom_emoji entities.
	CustomEmojiID string
	// Expandable reports whether a blockquote entity is expandable.
	Expandable bool
	// UnixTime is the Unix timestamp associated with date_time entities.
	UnixTime int
	// DateTimeFormat is the Telegram date-time format string for date_time entities.
	DateTimeFormat string
}

// ParseMessageEntity converts a single MTProto MessageEntityClass into a
// MessageEntity. Returns nil if raw is nil.
func ParseMessageEntity(raw tg.MessageEntityClass) *MessageEntity {
	return ParseMessageEntityWithUsers(raw, nil)
}

// ParseMessageEntityWithUsers converts a single MTProto MessageEntityClass into
// a MessageEntity and resolves text_mention users from the supplied users map
// when possible. Returns nil if raw is nil.
func ParseMessageEntityWithUsers(raw tg.MessageEntityClass, users map[int64]*tg.User) *MessageEntity {
	if raw == nil {
		return nil
	}
	e := &MessageEntity{}
	var userID int64
	switch r := raw.(type) {
	case *tg.MessageEntityMention:
		e.Type = MessageEntityTypeMention
		e.Offset, e.Length = int(r.Offset), int(r.Length)
	case *tg.MessageEntityHashtag:
		e.Type = MessageEntityTypeHashtag
		e.Offset, e.Length = int(r.Offset), int(r.Length)
	case *tg.MessageEntityCashtag:
		e.Type = MessageEntityTypeCashtag
		e.Offset, e.Length = int(r.Offset), int(r.Length)
	case *tg.MessageEntityBotCommand:
		e.Type = MessageEntityTypeBotCommand
		e.Offset, e.Length = int(r.Offset), int(r.Length)
	case *tg.MessageEntityURL:
		e.Type = MessageEntityTypeURL
		e.Offset, e.Length = int(r.Offset), int(r.Length)
	case *tg.MessageEntityEmail:
		e.Type = MessageEntityTypeEmail
		e.Offset, e.Length = int(r.Offset), int(r.Length)
	case *tg.MessageEntityPhone:
		e.Type = MessageEntityTypePhoneNumber
		e.Offset, e.Length = int(r.Offset), int(r.Length)
	case *tg.MessageEntityBold:
		e.Type = MessageEntityTypeBold
		e.Offset, e.Length = int(r.Offset), int(r.Length)
	case *tg.MessageEntityItalic:
		e.Type = MessageEntityTypeItalic
		e.Offset, e.Length = int(r.Offset), int(r.Length)
	case *tg.MessageEntityUnderline:
		e.Type = MessageEntityTypeUnderline
		e.Offset, e.Length = int(r.Offset), int(r.Length)
	case *tg.MessageEntityStrike:
		e.Type = MessageEntityTypeStrikethrough
		e.Offset, e.Length = int(r.Offset), int(r.Length)
	case *tg.MessageEntitySpoiler:
		e.Type = MessageEntityTypeSpoiler
		e.Offset, e.Length = int(r.Offset), int(r.Length)
	case *tg.MessageEntityCode:
		e.Type = MessageEntityTypeCode
		e.Offset, e.Length = int(r.Offset), int(r.Length)
	case *tg.MessageEntityPre:
		e.Type = MessageEntityTypePre
		e.Offset, e.Length = int(r.Offset), int(r.Length)
		e.Language = r.Language
	case *tg.MessageEntityBlockquote:
		e.Type = MessageEntityTypeBlockquote
		e.Offset, e.Length = int(r.Offset), int(r.Length)
		e.Expandable = r.Collapsed
	case *tg.MessageEntityTextURL:
		e.Type = MessageEntityTypeTextLink
		e.Offset, e.Length = int(r.Offset), int(r.Length)
		e.URL = r.URL
	case *tg.MessageEntityMentionName:
		e.Type = MessageEntityTypeTextMention
		e.Offset, e.Length = int(r.Offset), int(r.Length)
		userID = r.UserID
	case *tg.InputMessageEntityMentionName:
		e.Type = MessageEntityTypeTextMention
		e.Offset, e.Length = int(r.Offset), int(r.Length)
		userID = inputUserID(r.UserID)
	case *tg.MessageEntityBankCard:
		e.Type = MessageEntityTypeBankCard
		e.Offset, e.Length = int(r.Offset), int(r.Length)
	case *tg.MessageEntityCustomEmoji:
		e.Type = MessageEntityTypeCustomEmoji
		e.Offset, e.Length = int(r.Offset), int(r.Length)
		if r.DocumentID != 0 {
			e.CustomEmojiID = strconv.FormatInt(r.DocumentID, 10)
		}
	case *tg.MessageEntityFormattedDate:
		e.Type = MessageEntityTypeDateTime
		e.Offset, e.Length = int(r.Offset), int(r.Length)
		e.UnixTime = int(r.Date)
		e.DateTimeFormat = messageEntityDateTimeFormat(r)
	default:
		e.Type = MessageEntityTypeUnknown
		if base, ok := raw.(interface{ GetOffset() int32 }); ok {
			e.Offset = int(base.GetOffset())
		}
		if base, ok := raw.(interface{ GetLength() int32 }); ok {
			e.Length = int(base.GetLength())
		}
	}
	if userID != 0 {
		e.UserID = userID
		if users != nil {
			if user, ok := users[userID]; ok {
				e.User = ParseUser(user)
			}
		}
	}
	return e
}

// ParseMessageEntities converts a slice of MTProto MessageEntityClass values
// into a slice of MessageEntity. Returns nil if raw is nil.
func ParseMessageEntities(raw []tg.MessageEntityClass) []*MessageEntity {
	return ParseMessageEntitiesWithUsers(raw, nil)
}

// ParseMessageEntitiesWithUsers converts a slice of MTProto MessageEntityClass
// values into MessageEntity values and resolves text_mention users from the
// supplied users map when possible. Returns nil if raw is nil.
func ParseMessageEntitiesWithUsers(raw []tg.MessageEntityClass, users map[int64]*tg.User) []*MessageEntity {
	if raw == nil {
		return nil
	}
	result := make([]*MessageEntity, 0, len(raw))
	for _, r := range raw {
		if e := ParseMessageEntityWithUsers(r, users); e != nil {
			result = append(result, e)
		}
	}
	return result
}

func messageEntityDateTimeFormat(entity *tg.MessageEntityFormattedDate) string {
	if entity.Relative {
		return "r"
	}

	format := ""
	if entity.DayOfWeek {
		format += "w"
	}
	if entity.ShortDate {
		format += "d"
	} else if entity.LongDate {
		format += "D"
	}
	if entity.ShortTime {
		format += "t"
	} else if entity.LongTime {
		format += "T"
	}
	return format
}
