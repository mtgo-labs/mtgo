package parser

import (
	"regexp"
	"strconv"
	"strings"

	"github.com/mtgo-labs/mtgo/tg"
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

	tagRe := regexp.MustCompile(`<(/?)(\w+)([^>]*)>`)

	var result strings.Builder
	lastIdx := 0

	matches := tagRe.FindAllStringSubmatchIndex(text, -1)
	for _, loc := range matches {
		fullStart, fullEnd := loc[0], loc[1]
		result.WriteString(text[lastIdx:fullStart])

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
	result.WriteString(text[lastIdx:])

	cleaned := htmlUnescape(result.String())

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
			userID, _ := strconv.ParseInt(after, 10, 64)
			return &tg.InputMessageEntityMentionName{
				Offset: int32(offset),
				Length: int32(length),
				UserID: &tg.InputUser{UserID: userID},
			}
		}
		return &tg.MessageEntityTextURL{Offset: int32(offset), Length: int32(length), URL: href}
	}
	return nil
}

func parseAttrs(s string) map[string]string {
	attrs := make(map[string]string)
	re := regexp.MustCompile(`(\w+)="([^"]*)"`)
	matches := re.FindAllStringSubmatch(s, -1)
	for _, m := range matches {
		attrs[strings.ToLower(m[1])] = m[2]
	}
	return attrs
}

func htmlUnescape(s string) string {
	s = strings.ReplaceAll(s, "&amp;", "&")
	s = strings.ReplaceAll(s, "&lt;", "<")
	s = strings.ReplaceAll(s, "&gt;", ">")
	s = strings.ReplaceAll(s, "&quot;", "\"")
	return s
}

func adjustEntityOffsets(entities []tg.MessageEntityClass) []tg.MessageEntityClass {
	return entities
}
