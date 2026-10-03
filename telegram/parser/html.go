package parser

import (
	"regexp"
	"strconv"
	"strings"

	"github.com/mtgo-labs/mtgo/tg"
)

var (
	// htmlTagRe matches opening and closing HTML tags. The tag name group
	// accepts word characters and hyphens (e.g. tg-spoiler, tg-emoji).
	htmlTagRe = regexp.MustCompile(`<(/?)([\w-]+)([^>]*)>`)
	// htmlAttrRe matches key="value" pairs. Attribute names may contain
	// hyphens (e.g. emoji-id, class). Boolean attributes without a value
	// (e.g. expandable, collapsed) are handled separately.
	htmlAttrRe = regexp.MustCompile(`([\w-]+)="([^"]*)"`)
	// htmlBoolAttrRe matches boolean attributes: word characters and hyphens
	// that form a complete token. Anchored to start and end to prevent
	// false positives on stray fragments (e.g. "=value" matching "value").
	htmlBoolAttrRe = regexp.MustCompile(`^([\w-]+)$`)
	// preCodeOpenRe matches the documented <pre><code class="language-…">
	// opening pair used to declare a pre entity's programming language.
	preCodeOpenRe = regexp.MustCompile(`<pre>\s*<code class="language-([\w#+.-]*)">`)
	// preCodeCloseRe matches the closing </code></pre> pair produced by the
	// opening pair above.
	preCodeCloseRe = regexp.MustCompile(`</code>\s*</pre>`)
	// numericEntityRe matches decimal and hexadecimal numerical HTML
	// entities (&#38; / &#x26;).
	numericEntityRe = regexp.MustCompile(`&#(x?[0-9a-fA-F]+);`)
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
// Entity offsets and lengths are measured in UTF-16 code units, as required by
// the Telegram API.
func (p *HTMLParser) Parse(html string) (string, []tg.MessageEntityClass, error) {
	text := normalizePreCodeLanguage(html)
	var entities []tg.MessageEntityClass
	var stack []htmlTag

	var result strings.Builder
	// The result stays plain UTF-8 end to end; a running UTF-16 counter drives
	// all entity offsets because Telegram positions entities in UTF-16 code
	// units, which coincide with byte offsets only for pure-ASCII text.
	utf16Len := 0
	lastIdx := 0

	matches := htmlTagRe.FindAllStringSubmatchIndex(text, -1)
	for _, loc := range matches {
		fullStart, fullEnd := loc[0], loc[1]
		// Unescape each text fragment as it is emitted so that entity offsets
		// (measured via utf16Len) refer to the final, unescaped text. Doing
		// the unescape after building the whole string would shift offsets
		// wherever an HTML entity appears before a formatted region.
		fragment := htmlUnescape(text[lastIdx:fullStart])
		result.WriteString(fragment)
		utf16Len += utf16Length(fragment)

		closing := text[loc[2]:loc[3]] == "/"
		tagName := strings.ToLower(text[loc[4]:loc[5]])
		attrStr := text[loc[6]:loc[7]]

		if closing {
			for i := len(stack) - 1; i >= 0; i-- {
				if stack[i].tag == tagName {
					ent := p.createEntity(stack[i], utf16Len)
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
				offset: utf16Len,
				attrs:  attrs,
			})
		}

		lastIdx = fullEnd
	}
	tail := htmlUnescape(text[lastIdx:])
	result.WriteString(tail)

	entities = dropEmptyEntities(entities)

	return result.String(), entities, nil
}

// dropEmptyEntities removes zero-length entities. They carry no user-visible
// formatting and only appear as artifacts of delimiter pairs like "**text**"
// under the single-asterisk MarkdownV2 grammar.
func dropEmptyEntities(entities []tg.MessageEntityClass) []tg.MessageEntityClass {
	if len(entities) == 0 {
		return nil
	}
	kept := entities[:0]
	for _, e := range entities {
		switch v := e.(type) {
		case *tg.MessageEntityBold:
			if v.Length == 0 {
				continue
			}
		case *tg.MessageEntityItalic:
			if v.Length == 0 {
				continue
			}
		case *tg.MessageEntityUnderline:
			if v.Length == 0 {
				continue
			}
		case *tg.MessageEntityStrike:
			if v.Length == 0 {
				continue
			}
		case *tg.MessageEntitySpoiler:
			if v.Length == 0 {
				continue
			}
		}
		kept = append(kept, e)
	}
	if len(kept) == 0 {
		return nil
	}
	return kept
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
	case "spoiler", "tg-spoiler":
		return &tg.MessageEntitySpoiler{Offset: int32(offset), Length: int32(length)}
	case "span":
		// <span class="tg-spoiler"> is the canonical Bot API spoiler tag.
		if tag.attrs != nil && tag.attrs["class"] == "tg-spoiler" {
			return &tg.MessageEntitySpoiler{Offset: int32(offset), Length: int32(length)}
		}
		return nil
	case "tg-emoji", "emoji":
		emojiID := int64(0)
		if tag.attrs != nil {
			if idStr := tag.attrs["emoji-id"]; idStr != "" {
				if id, err := strconv.ParseInt(idStr, 10, 64); err == nil {
					emojiID = id
				}
			}
		}
		if emojiID == 0 {
			return nil
		}
		return &tg.MessageEntityCustomEmoji{Offset: int32(offset), Length: int32(length), DocumentID: emojiID}
	case "tg-time":
		// <tg-time unix="1647531900" format="wDT">22:45 tomorrow</tg-time>
		if tag.attrs == nil {
			return nil
		}
		unix := tag.attrs["unix"]
		if unix == "" || !unixTimeRE.MatchString(unix) {
			return nil
		}
		date, err := strconv.ParseInt(unix, 10, 32)
		if err != nil || date <= 0 {
			return nil
		}
		format := tag.attrs["format"]
		relative, shortTime, longTime, shortDate, longDate, dayOfWeek, ok := parseDateTimeFormat(format)
		if !ok {
			return nil
		}
		return &tg.MessageEntityFormattedDate{
			Relative:  relative,
			ShortTime: shortTime,
			LongTime:  longTime,
			ShortDate: shortDate,
			LongDate:  longDate,
			DayOfWeek: dayOfWeek,
			Offset:    int32(offset),
			Length:    int32(length),
			Date:      int32(date),
		}
	case "blockquote":
		collapsed := false
		if tag.attrs != nil {
			_, collapsed = tag.attrs["collapsed"]
			if !collapsed {
				_, collapsed = tag.attrs["expandable"]
			}
		}
		return &tg.MessageEntityBlockquote{Offset: int32(offset), Length: int32(length), Collapsed: collapsed}
	case "a":
		href := ""
		if tag.attrs != nil {
			href = tag.attrs["href"]
		}
		// Blocklist dangerous URL protocols.
		if isDangerousURL(href) {
			return nil
		}
		// mailto: links → MessageEntityEmail.
		if _, ok := strings.CutPrefix(href, "mailto:"); ok {
			return &tg.MessageEntityEmail{Offset: int32(offset), Length: int32(length)}
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
		if href == "" {
			return nil
		}
		return &tg.MessageEntityTextURL{Offset: int32(offset), Length: int32(length), URL: href}
	}
	return nil
}

// isDangerousURL checks whether a URL uses a potentially dangerous protocol.
func isDangerousURL(url string) bool {
	lower := strings.ToLower(url)
	for _, proto := range dangerousURLProtocols {
		if strings.HasPrefix(lower, proto) {
			return true
		}
	}
	return false
}

// dangerousURLProtocols lists URL protocols that are rejected in href attributes.
var dangerousURLProtocols = []string{
	"javascript:",
	"data:",
	"vbscript:",
	"file:",
}

func parseAttrs(s string) map[string]string {
	attrs := make(map[string]string)
	// Key="value" pairs (e.g. href="https://example.com", emoji-id="12345").
	matches := htmlAttrRe.FindAllStringSubmatch(s, -1)
	for _, m := range matches {
		attrs[strings.ToLower(m[1])] = m[2]
	}
	// Boolean attributes: bare tokens without a value (e.g. expandable, collapsed).
	// Remove all key="value" spans first, then tokenise the remainder.
	remainder := htmlAttrRe.ReplaceAllString(s, "")
	for _, token := range strings.Fields(remainder) {
		// Accept word characters and hyphens only (reject stray =, quotes, etc.).
		if htmlBoolAttrRe.MatchString(token) {
			attrs[strings.ToLower(token)] = ""
		}
	}
	return attrs
}

func htmlUnescape(s string) string {
	// Replace &amp; last: unescaping it first would turn "&amp;lt;" into "&lt;"
	// and then into "<", losing a level of escaping.
	s = strings.ReplaceAll(s, "&lt;", "<")
	s = strings.ReplaceAll(s, "&gt;", ">")
	s = strings.ReplaceAll(s, "&quot;", "\"")
	s = numericEntityRe.ReplaceAllStringFunc(s, func(m string) string {
		digits := strings.TrimSuffix(strings.TrimPrefix(m, "&#"), ";")
		base := 10
		if digits[0] == 'x' || digits[0] == 'X' {
			base = 16
			digits = digits[1:]
		}
		code, err := strconv.ParseInt(digits, base, 32)
		if err != nil || code == 0 {
			return m
		}
		return string(rune(code))
	})
	s = strings.ReplaceAll(s, "&amp;", "&")
	return s
}

// normalizePreCodeLanguage rewrites the documented
// <pre><code class="language-…">…</code></pre> pair into the internal
// <pre language="…">…</pre> form so a single pre entity with a language is
// produced. Standalone code tags (no pre parent) keep their plain-code
// semantics: the specification forbids languages on standalone code tags.
func normalizePreCodeLanguage(s string) string {
	if !strings.Contains(s, "<pre>") {
		return s
	}
	s = preCodeOpenRe.ReplaceAllString(s, `<pre language="$1">`)
	if !strings.Contains(s, `</code>`) {
		return s
	}
	return preCodeCloseRe.ReplaceAllString(s, "</pre>")
}
