package types

import "github.com/mtgo-labs/mtgo/tg"

// MessageEntity represents a formatting or semantic entity within a message text,
// such as bold, italic, mention, URL, or code blocks. Each entity is defined by
// its type, byte offset, and length within the message text.
type MessageEntity struct {
	// Type identifies the kind of entity (bold, italic, mention, URL, etc.).
	Type MessageEntityType
	// Offset is the zero-based byte offset where the entity starts in the message text.
	Offset int
	// Length is the number of bytes the entity spans in the message text.
	Length int
	// URL is the target URL for text_link entities, or empty for other types.
	URL string
	// UserID is the Telegram user ID for text_mention entities that reference a user
	// without a username, or zero for other types.
	UserID int64
	// Language is the programming language for pre entities, or empty for other types.
	Language string
}

// ParseMessageEntity converts a single MTProto MessageEntityClass into a
// MessageEntity. Returns nil if raw is nil.
func ParseMessageEntity(raw tg.MessageEntityClass) *MessageEntity {
	if raw == nil {
		return nil
	}
	e := &MessageEntity{}
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
	case *tg.MessageEntityTextURL:
		e.Type = MessageEntityTypeTextLink
		e.Offset, e.Length = int(r.Offset), int(r.Length)
		e.URL = r.URL
	case *tg.MessageEntityMentionName:
		e.Type = MessageEntityTypeTextMention
		e.Offset, e.Length = int(r.Offset), int(r.Length)
		e.UserID = r.UserID
	case *tg.MessageEntityBankCard:
		e.Type = MessageEntityTypeBankCard
		e.Offset, e.Length = int(r.Offset), int(r.Length)
	case *tg.MessageEntityCustomEmoji:
		e.Type = MessageEntityTypeCustomEmoji
		e.Offset, e.Length = int(r.Offset), int(r.Length)
	case *tg.MessageEntityFormattedDate:
		e.Type = MessageEntityTypeDateTime
		e.Offset, e.Length = int(r.Offset), int(r.Length)
	default:
		e.Type = MessageEntityTypeUnknown
		if base, ok := raw.(interface{ GetOffset() int32 }); ok {
			e.Offset = int(base.GetOffset())
		}
		if base, ok := raw.(interface{ GetLength() int32 }); ok {
			e.Length = int(base.GetLength())
		}
	}
	return e
}

// ParseMessageEntities converts a slice of MTProto MessageEntityClass values
// into a slice of MessageEntity. Returns nil if raw is nil.
func ParseMessageEntities(raw []tg.MessageEntityClass) []*MessageEntity {
	if raw == nil {
		return nil
	}
	result := make([]*MessageEntity, 0, len(raw))
	for _, r := range raw {
		if e := ParseMessageEntity(r); e != nil {
			result = append(result, e)
		}
	}
	return result
}
