package parser

import (
	"regexp"
	"strconv"
	"strings"

	"github.com/mtgo-labs/mtgo/tg"
)

var (
	htmlTagRe  = regexp.MustCompile(`<(/?)(\w+)([^>]*)>`)
	htmlAttrRe = regexp.MustCompile(`(\w+)="([^"]*)"`)
)

// HTMLParser parses Telegram HTML markup into plain text and message entities.
type HTMLParser struct{}

// NewHTMLParser returns a new HTMLParser ready for use.
func NewHTMLParser() *HTMLParser {
	return &HTMLParser{}
}

type htmlTag struct {
	tag    string
	offset int
	attrs  map[string]string
}

// Parse parses the given HTML string, strips all tags, and returns the plain text
// together with a slice of Telegram message entities representing the formatting.
// It returns an error if surrogate decoding fails.
func (p *HTMLParser) Parse(html string) (string, []tg.MessageEntityClass, error) {
	text := AddSurrogates(html)
	var entities []tg.MessageEntityClass
	var stack []htmlTag

	var result strings.Builder
	lastIdx := 0

	matches := htmlTagRe.FindAllStringSubmatchIndex(text, -1)
	for _, loc := range matches {
		fullStart, fullEnd := loc[0], loc[1]
		// Unescape each text fragment as it is emitted so that entity offsets
		// (measured via result.Len()) refer to the final, unescaped text. Doing
		// the unescape after building the whole string would shift offsets
		// wherever an HTML entity appears before a formatted region.
		result.WriteString(htmlUnescape(text[lastIdx:fullStart]))

		closing := text[loc[2]:loc[3]] == "/"
		tagName := strings.ToLower(text[loc[4]:loc[5]])
		attrStr := text[loc[6]:loc[7]]

		if closing {
			for i := len(stack) - 1; i >= 0; i-- {
				if stack[i].tag == tagName {
					ent := p.createEntity(stack[i], result.Len())
					if ent != nil {
						entities = append(entities, ent)
					}
					stack = append(stack[:i], stack[i+1:]...)
					break
				}
			}
		} else {
			attrs := parseAttrs(attrStr)
			stack = append(stack, htmlTag{
				tag:    tagName,
				offset: result.Len(),
				attrs:  attrs,
			})
		}

		lastIdx = fullEnd
	}
	result.WriteString(htmlUnescape(text[lastIdx:]))

	cleaned := result.String()

	finalText, err := RemoveSurrogates(cleaned)
	if err != nil {
		return "", nil, err
	}

	entities = adjustEntityOffsets(entities)

	return finalText, entities, nil
}

func (p *HTMLParser) createEntity(tag htmlTag, endOffset int) tg.MessageEntityClass {
	length := endOffset - tag.offset
	offset := tag.offset

	switch tag.tag {
	case "b", "strong":
		return &tg.MessageEntityBold{Offset: int32(offset), Length: int32(length)}
	case "i", "em":
		return &tg.MessageEntityItalic{Offset: int32(offset), Length: int32(length)}
	case "u", "ins":
		return &tg.MessageEntityUnderline{Offset: int32(offset), Length: int32(length)}
	case "s", "strike", "del":
		return &tg.MessageEntityStrike{Offset: int32(offset), Length: int32(length)}
	case "code":
		return &tg.MessageEntityCode{Offset: int32(offset), Length: int32(length)}
	case "pre":
		lang := ""
		if tag.attrs != nil {
			lang = tag.attrs["language"]
		}
		return &tg.MessageEntityPre{Offset: int32(offset), Length: int32(length), Language: lang}
	case "spoiler":
		return &tg.MessageEntitySpoiler{Offset: int32(offset), Length: int32(length)}
	case "blockquote":
		return &tg.MessageEntityBlockquote{Offset: int32(offset), Length: int32(length)}
	case "a":
		href := ""
		if tag.attrs != nil {
			href = tag.attrs["href"]
		}
		if after, ok := strings.CutPrefix(href, "tg://user?id="); ok {
			// Only emit a mention for a well-formed, positive user id. A bogus or
			// attacker-supplied id (e.g. from re-parsed untrusted HTML) falls back
			// to a plain text URL rather than a forged mention of an arbitrary user.
			if userID, perr := strconv.ParseInt(after, 10, 64); perr == nil && userID > 0 {
				return &tg.InputMessageEntityMentionName{
					Offset: int32(offset),
					Length: int32(length),
					UserID: &tg.InputUser{UserID: userID},
				}
			}
			return &tg.MessageEntityTextURL{Offset: int32(offset), Length: int32(length), URL: href}
		}
		return &tg.MessageEntityTextURL{Offset: int32(offset), Length: int32(length), URL: href}
	}
	return nil
}

func parseAttrs(s string) map[string]string {
	attrs := make(map[string]string)
	matches := htmlAttrRe.FindAllStringSubmatch(s, -1)
	for _, m := range matches {
		attrs[strings.ToLower(m[1])] = m[2]
	}
	return attrs
}

func htmlUnescape(s string) string {
	// Replace &amp; last: unescaping it first would turn "&amp;lt;" into "&lt;"
	// and then into "<", losing a level of escaping.
	s = strings.ReplaceAll(s, "&lt;", "<")
	s = strings.ReplaceAll(s, "&gt;", ">")
	s = strings.ReplaceAll(s, "&quot;", "\"")
	s = strings.ReplaceAll(s, "&amp;", "&")
	return s
}

// adjustEntityOffsets is intentionally a no-op. Astral code points are encoded
// by AddSurrogates as exactly 4 bytes (two surrogate units), matching the 4-byte
// UTF-8 encoding RemoveSurrogates emits, so byte offsets are length-preserving
// across the surrogate round-trip. The text fragments are already unescaped as
// they are written, so no further offset translation is needed.
func adjustEntityOffsets(entities []tg.MessageEntityClass) []tg.MessageEntityClass {
	return entities
}
