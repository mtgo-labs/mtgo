package parser

import (
	"fmt"
	"strings"
	"unicode/utf16"
	"unicode/utf8"
)

// AddSurrogates encodes Unicode code points above U+FFFF as UTF-16 surrogate pairs,
// so that entity offsets match Telegram's UTF-16-based positioning.
//
// The parsers no longer use this; it remains for callers that need the
// round-trip with RemoveSurrogates.
func AddSurrogates(text string) string {
	var b strings.Builder
	b.Grow(len(text) + len(text)/2)
	for _, r := range text {
		if r > 0xFFFF {
			r1, r2 := utf16.EncodeRune(r)
			b.WriteByte(byte(r1))
			b.WriteByte(byte(r1 >> 8))
			b.WriteByte(byte(r2))
			b.WriteByte(byte(r2 >> 8))
		} else {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// RemoveSurrogates decodes UTF-16 surrogate pairs encoded by AddSurrogates back
// into their original Unicode code points, returning valid UTF-8 text.
//
// Valid UTF-8 text is returned unchanged: Arabic, Hebrew, and other multi-byte
// scripts use lead bytes 0xD8–0xDF that collide with the UTF-16 surrogate
// ranges when scanned as raw bytes, and AddSurrogates output (a lone 0xD8–0xDF
// byte can never be a UTF-8 continuation) is invalid UTF-8 by construction, so
// whole-string validity disambiguates the two. An error is returned only for
// invalid UTF-8 that also contains an unmatched surrogate-looking window.
func RemoveSurrogates(text string) (string, error) {
	if utf8.ValidString(text) {
		return text, nil
	}
	data := []byte(text)
	var result strings.Builder
	i := 0
	for i < len(data) {
		if i+3 < len(data) {
			lo := uint16(data[i]) | uint16(data[i+1])<<8
			if lo >= 0xD800 && lo <= 0xDBFF {
				hi := uint16(data[i+2]) | uint16(data[i+3])<<8
				if hi >= 0xDC00 && hi <= 0xDFFF {
					r := utf16.DecodeRune(rune(lo), rune(hi))
					result.WriteRune(r)
					i += 4
					continue
				}
				return "", fmt.Errorf("invalid surrogate pair at position %d", i)
			}
		}
		if data[i] < 0x80 {
			result.WriteByte(data[i])
			i++
		} else {
			_, size := utf8.DecodeRune(data[i:])
			result.Write(data[i : i+size])
			i += size
		}
	}
	return result.String(), nil
}

// utf16Length returns the length of s in UTF-16 code units: one unit per code
// point in the Basic Multilingual Plane, two per astral code point. Telegram
// measures message entity offsets and lengths in these units.
func utf16Length(s string) int {
	units := 0
	for _, r := range s {
		if r > 0xFFFF {
			units += 2
		} else {
			units++
		}
	}
	return units
}

// ReplaceOnce replaces the first occurrence of old with newStr in the portion of
// source starting at the given start index, leaving the prefix before start unchanged.
func ReplaceOnce(source, old, newStr string, start int) string {
	return source[:start] + strings.Replace(source[start:], old, newStr, 1)
}
