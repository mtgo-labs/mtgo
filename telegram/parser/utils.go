package parser

import (
	"fmt"
	"strings"
	"unicode/utf16"
	"unicode/utf8"
)

// AddSurrogates encodes Unicode code points above U+FFFF as UTF-16 surrogate pairs,
// so that entity offsets match Telegram's UTF-16-based positioning.
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

// RemoveSurrogates decodes UTF-16 surrogate pairs back into their original
// Unicode code points, returning valid UTF-8 text.
// It returns an error if an unmatched or invalid surrogate pair is found.
func RemoveSurrogates(text string) (string, error) {
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

// ReplaceOnce replaces the first occurrence of old with newStr in the portion of
// source starting at the given start index, leaving the prefix before start unchanged.
func ReplaceOnce(source, old, newStr string, start int) string {
	return source[:start] + strings.Replace(source[start:], old, newStr, 1)
}
