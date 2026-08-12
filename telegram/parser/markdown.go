package parser

import (
	"fmt"
	"regexp"
	"strings"

	tl "github.com/mtgo-labs/mtgo/tg"
)

// MarkdownParser parses Telegram MarkdownV2-formatted text into plain text
// and message entities by converting to HTML and delegating to HTMLParser.
type MarkdownParser struct{}

// NewMarkdownParser returns a new MarkdownParser ready for use.
func NewMarkdownParser() *MarkdownParser {
	return &MarkdownParser{}
}

// Parse converts MarkdownV2-formatted text to HTML and delegates to HTMLParser.
// It returns the plain text and corresponding Telegram message entities.
func (p *MarkdownParser) Parse(md string) (string, []tl.MessageEntityClass, error) {
	html := mdToHTML(md)
	return htmlParser.Parse(html)
}

// linkRe matches MarkdownV2 [text](url) links.
var linkRe = regexp.MustCompile(`\[([^\]]*)\]\(([^)]+)\)`)

func mdToHTML(md string) string {
	s := md

	// 1a. Pre-escape: handle \\ and \` before code extraction.
	// These two sequences affect code boundary detection — if
	// processed later, \` would be misread as a code delimiter and
	// \\ would prevent \` from being recognised as an escaped backtick.
	s = escapePre(s)

	// 2. Extract code spans (``` and `) and replace with numbered placeholders
	//    so that formatting delimiters inside code are not processed. Must run
	//    before blockquote processing so that > inside code blocks is literal.
	codeBlocks := make([]string, 0, 8)
	s = extractCodeBlocks(s, &codeBlocks)
	s = extractCodeSpans(s, &codeBlocks)

	// 1b. Post-escape: handle remaining escape sequences. These run after
	//     code extraction so that \* inside `code` is preserved literally.
	s = escapePost(s)

	// 3. Blockquotes — > and >>> at line start. Must run after code extraction
	//    (so > inside code is protected) and before inline formatting (so **bold**
	//    inside a blockquote is still processed).
	s = processBlockquotes(s)

	// 4. Links [text](url) — run before formatting delimiters so that ** inside
	//    [text] sections is not prematurely consumed.
	s = replaceLinks(s)

	// 5. Formatting delimiters — longer delimiters first to avoid partial
	//    consumption (e.g. "**" before "*").
	s = replaceDelimited(s, "**", "<b>", "</b>")              // bold
	s = replaceDelimited(s, "__", "<u>", "</u>")              // underline (MarkdownV2)
	s = replaceDelimited(s, "~~", "<s>", "</s>")              // strikethrough
	s = replaceDelimited(s, "||", "<spoiler>", "</spoiler>") // spoiler
	s = replaceDelimited(s, "*", "<i>", "</i>")               // italic
	s = replaceDelimited(s, "_", "<i>", "</i>")               // italic (alternative)

	// 6. Restore code spans from placeholders.
	s = restoreCodeBlocks(s, codeBlocks)

	// 7. Restore escaped characters, HTML-escaping <, >, and & so the
	//    HTML parser treats them as literal text. Handles both pre-escape
	//    and post-escape placeholders.
	s = unescapeAll(s)

	return s
}

// codeBlockPlaceholderFmt is the format string for code block placeholders.
const codeBlockPlaceholderFmt = "\uE100%04d\uE100"

// codeBlockPlaceholderRe matches code block placeholders.
var codeBlockPlaceholderRe = regexp.MustCompile("\uE100(\\d{4})\uE100")

// extractCodeBlocks replaces fenced code blocks (```) with placeholders
// and stores the original content (including delimiters converted to HTML).
func extractCodeBlocks(s string, blocks *[]string) string {
	return replaceDelimitedWithCollect(s, "```", func(content string) string {
		idx := len(*blocks)
		// Extract language hint from the first line if present.
		lang := ""
		rest := content
		if nl := strings.IndexByte(content, '\n'); nl >= 0 {
			firstLine := strings.TrimSpace(content[:nl])
			// A language hint is a single word without spaces.
			if !strings.ContainsAny(firstLine, " \t") && firstLine != "" {
				lang = firstLine
				rest = content[nl+1:]
			}
		}
		if lang != "" {
			*blocks = append(*blocks, "<pre language=\""+lang+"\">"+rest+"</pre>")
		} else {
			*blocks = append(*blocks, "<pre>"+rest+"</pre>")
		}
		return fmt.Sprintf(codeBlockPlaceholderFmt, idx)
	})
}

// extractCodeSpans replaces inline code (`) with placeholders.
func extractCodeSpans(s string, blocks *[]string) string {
	return replaceDelimitedWithCollect(s, "`", func(content string) string {
		idx := len(*blocks)
		*blocks = append(*blocks, "<code>"+content+"</code>")
		return fmt.Sprintf(codeBlockPlaceholderFmt, idx)
	})
}

// restoreCodeBlocks replaces code block placeholders with their stored HTML.
func restoreCodeBlocks(s string, blocks []string) string {
	return codeBlockPlaceholderRe.ReplaceAllStringFunc(s, func(match string) string {
		parts := codeBlockPlaceholderRe.FindStringSubmatch(match)
		if len(parts) != 2 {
			return match
		}
		var idx int
		fmt.Sscanf(parts[1], "%04d", &idx)
		if idx < 0 || idx >= len(blocks) {
			return match
		}
		return blocks[idx]
	})
}

// replaceDelimitedWithCollect is like replaceDelimited but collects the
// content between delimiter pairs and passes each to the collect function,
// replacing the entire delimited span with the function's return value.
func replaceDelimitedWithCollect(s, delim string, collect func(content string) string) string {
	var b strings.Builder
	b.Grow(len(s))
	for {
		idx := strings.Index(s, delim)
		if idx == -1 {
			b.WriteString(s)
			break
		}
		b.WriteString(s[:idx])
		s = s[idx+len(delim):]

		// Find the closing delimiter.
		end := strings.Index(s, delim)
		if end == -1 {
			// No closing delimiter — treat the rest as literal text.
			b.WriteString(delim)
			b.WriteString(s)
			break
		}
		content := s[:end]
		b.WriteString(collect(content))
		s = s[end+len(delim):]
	}
	return b.String()
}

// replaceLinks converts MarkdownV2 [text](url) syntax to HTML <a> tags.
// Characters that could break HTML parsing (" and >) are percent-encoded.
func replaceLinks(s string) string {
	return linkRe.ReplaceAllStringFunc(s, func(match string) string {
		parts := linkRe.FindStringSubmatch(match)
		if len(parts) != 3 {
			return match
		}
		text := parts[1]
		url := parts[2]
		// Prevent URL content from breaking the HTML attribute or tag
		// boundary. Per RFC 3986 these characters should already be
		// percent-encoded in valid URLs, but defensive escaping costs
		// nothing and prevents malformed-output edge cases.
		url = strings.ReplaceAll(url, `"`, `%22`)
		url = strings.ReplaceAll(url, `>`, `%3E`)
		return `<a href="` + url + `">` + text + `</a>`
	})
}

// processBlockquotes converts MarkdownV2 line-prefix blockquote syntax to
// HTML <blockquote> tags. > at line start creates a plain blockquote;
// >>> creates an expandable blockquote. An optional space may follow the
// prefix. Lines without a prefix are passed through unchanged.
//
// Must run after code extraction so that > inside code spans is already
// placeholder-protected.
func processBlockquotes(s string) string {
	lines := strings.Split(s, "\n")
	for i, line := range lines {
		if trimmed, ok := strings.CutPrefix(line, ">>>"); ok {
			trimmed = strings.TrimPrefix(trimmed, " ")
			lines[i] = "<blockquote expandable>" + trimmed + "</blockquote>"
		} else if trimmed, ok := strings.CutPrefix(line, ">"); ok {
			trimmed = strings.TrimPrefix(trimmed, " ")
			lines[i] = "<blockquote>" + trimmed + "</blockquote>"
		}
	}
	return strings.Join(lines, "\n")
}

// replaceDelimited performs a blind state-machine replacement: it alternates
// between open and close tags on each occurrence of delim in s. This is
// correct for non-nested delimiters; nested formatting is handled by the
// HTML-to-entity conversion layer (HTMLParser).
func replaceDelimited(s, delim, openTag, closeTag string) string {
	var b strings.Builder
	b.Grow(len(s))
	open := true
	for {
		idx := strings.Index(s, delim)
		if idx == -1 {
			b.WriteString(s)
			break
		}
		b.WriteString(s[:idx])
		if open {
			b.WriteString(openTag)
		} else {
			b.WriteString(closeTag)
		}
		s = s[idx+len(delim):]
		open = !open
	}
	return b.String()
}

// MarkdownV2 escape sequences. Each backslash-character sequence is
// replaced with a Unicode private-use placeholder during the initial
// pass, then restored to its literal character after all formatting.
//
// Split into two groups:
//   - preEscapes: \\ and \` — processed before code extraction because
//     they affect code boundary detection.
//   - postEscapes: remaining 16 sequences — processed after code
//     extraction so that escapes inside code spans are preserved literally.
//
// Within each group \\ is ordered first so that the backslash in \\X is
// not consumed by the \X handler.
var preEscapes = []struct {
	escaped     string
	placeholder string
	literal     string
}{
	{`\\`, "\uE000", `\`},
	{"\\`", "\uE003", "`"},
}

var postEscapes = []struct {
	escaped     string
	placeholder string
	literal     string
}{
	{`\*`, "\uE001", `*`},
	{`\_`, "\uE002", `_`},
	{`\~`, "\uE004", `~`},
	{`\|`, "\uE005", `|`},
	{`\[`, "\uE006", `[`},
	{`\]`, "\uE007", `]`},
	{`\(`, "\uE008", `(`},
	{`\)`, "\uE009", `)`},
	{`\{`, "\uE00A", `{`},
	{`\}`, "\uE00B", `}`},
	{`\<`, "\uE00C", `&lt;`},
	{`\>`, "\uE00D", `&gt;`},
	{`\!`, "\uE00E", `!`},
	{`\.`, "\uE00F", `.`},
	{`\-`, "\uE010", `-`},
	{`\=`, "\uE011", `=`},
	{`\+`, "\uE012", `+`},
	{`\#`, "\uE013", `#`},
}

// allEscapes is the concatenation of preEscapes and postEscapes, used
// by unescapeAll to restore all placeholders in a single pass.
var allEscapes []struct {
	placeholder string
	literal     string
}

func init() {
	allEscapes = make([]struct {
		placeholder string
		literal     string
	}, len(preEscapes)+len(postEscapes))
	i := 0
	for _, p := range preEscapes {
		allEscapes[i] = struct {
			placeholder string
			literal     string
		}{p.placeholder, p.literal}
		i++
	}
	for _, p := range postEscapes {
		allEscapes[i] = struct {
			placeholder string
			literal     string
		}{p.placeholder, p.literal}
		i++
	}
}

func escapePre(s string) string {
	if !strings.Contains(s, `\`) {
		return s
	}
	for _, p := range preEscapes {
		s = strings.ReplaceAll(s, p.escaped, p.placeholder)
	}
	return s
}

func escapePost(s string) string {
	if !strings.Contains(s, `\`) {
		return s
	}
	for _, p := range postEscapes {
		s = strings.ReplaceAll(s, p.escaped, p.placeholder)
	}
	return s
}

func unescapeAll(s string) string {
	// Fast path: all placeholders share the UTF-8 0xEE 0x80 prefix.
	if !strings.Contains(s, "\xEE\x80") {
		return s
	}
	for _, p := range allEscapes {
		s = strings.ReplaceAll(s, p.placeholder, p.literal)
	}
	return s
}
