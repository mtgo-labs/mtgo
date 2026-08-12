package parser

import (
	"reflect"
	"testing"

	tl "github.com/mtgo-labs/mtgo/tg"
)

func TestAddSurrogates_Basic(t *testing.T) {
	got := AddSurrogates("hello")
	if got != "hello" {
		t.Errorf("expected hello, got %q", got)
	}
}

func TestAddSurrogates_SMP(t *testing.T) {
	got := AddSurrogates("🎉")
	if len(got) != 4 {
		t.Errorf("expected length 4, got %d", len(got))
	}
}

func TestRemoveSurrogates_RoundTrip(t *testing.T) {
	original := "🎉party🎉"
	surrogated := AddSurrogates(original)
	restored, err := RemoveSurrogates(surrogated)
	if err != nil {
		t.Fatal(err)
	}
	if restored != original {
		t.Errorf("round trip: got %q, want %q", restored, original)
	}
}

func TestReplaceOnce(t *testing.T) {
	got := ReplaceOnce("hello world hello", "hello", "HI", 0)
	if got != "HI world hello" {
		t.Errorf("got %q", got)
	}
}

func TestHTMLParser_Bold(t *testing.T) {
	p := NewHTMLParser()
	text, entities, err := p.Parse("<b>hello</b>")
	if err != nil {
		t.Fatal(err)
	}
	if text != "hello" {
		t.Errorf("text = %q", text)
	}
	if len(entities) != 1 {
		t.Fatalf("entities = %d, want 1", len(entities))
	}
	if _, ok := entities[0].(*tl.MessageEntityBold); !ok {
		t.Errorf("expected MessageEntityBold, got %T", entities[0])
	}
}

func TestHTMLParser_Italic(t *testing.T) {
	p := NewHTMLParser()
	text, entities, err := p.Parse("<i>world</i>")
	if err != nil {
		t.Fatal(err)
	}
	if text != "world" {
		t.Errorf("text = %q", text)
	}
	if len(entities) != 1 {
		t.Fatal("expected 1 entity")
	}
	if _, ok := entities[0].(*tl.MessageEntityItalic); !ok {
		t.Errorf("expected MessageEntityItalic, got %T", entities[0])
	}
}

func TestHTMLParser_Code(t *testing.T) {
	p := NewHTMLParser()
	_, entities, err := p.Parse("<code>x := 1</code>")
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := entities[0].(*tl.MessageEntityCode); !ok {
		t.Errorf("expected MessageEntityCode, got %T", entities[0])
	}
}

func TestHTMLParser_TextURL(t *testing.T) {
	p := NewHTMLParser()
	_, entities, err := p.Parse(`<a href="https://example.com">click</a>`)
	if err != nil {
		t.Fatal(err)
	}
	ent, ok := entities[0].(*tl.MessageEntityTextURL)
	if !ok {
		t.Fatalf("expected MessageEntityTextURL, got %T", entities[0])
	}
	if ent.URL != "https://example.com" {
		t.Errorf("URL = %q", ent.URL)
	}
}

// TestHTMLParser_OffsetsAfterEntity verifies that entity offsets remain correct
// when an HTML-escaped character (which shrinks during unescaping) precedes a
// formatted region. Regression test for the htmlUnescape offset bug.
func TestHTMLParser_OffsetsAfterEntity(t *testing.T) {
	p := NewHTMLParser()
	text, entities, err := p.Parse(`<b>bold</b> &amp; <i>italic</i>`)
	if err != nil {
		t.Fatal(err)
	}
	if want := "bold & italic"; text != want {
		t.Fatalf("text = %q, want %q", text, want)
	}
	if len(entities) != 2 {
		t.Fatalf("entities = %d, want 2", len(entities))
	}
	bold, ok := entities[0].(*tl.MessageEntityBold)
	if !ok {
		t.Fatalf("expected MessageEntityBold, got %T", entities[0])
	}
	if bold.Offset != 0 || bold.Length != 4 {
		t.Errorf("bold = {Offset:%d, Length:%d}, want {0, 4}", bold.Offset, bold.Length)
	}
	ital, ok := entities[1].(*tl.MessageEntityItalic)
	if !ok {
		t.Fatalf("expected MessageEntityItalic, got %T", entities[1])
	}
	// "bold & " is 7 bytes, so italic starts at offset 7 and spans 6 bytes.
	if ital.Offset != 7 || ital.Length != 6 {
		t.Errorf("italic = {Offset:%d, Length:%d}, want {7, 6}", ital.Offset, ital.Length)
	}
}

// TestHTMLParser_MentionNameValidation verifies that a malformed or
// non-positive tg://user?id= falls back to a TextURL instead of a forged
// mention of an arbitrary user id.
func TestHTMLParser_MentionNameValidation(t *testing.T) {
	p := NewHTMLParser()

	// Valid positive id → mention.
	_, entities, err := p.Parse(`<a href="tg://user?id=12345">u</a>`)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := entities[0].(*tl.InputMessageEntityMentionName); !ok {
		t.Errorf("expected MentionName for valid id, got %T", entities[0])
	}

	// Invalid id → TextURL fallback (no forged mention).
	_, entities, err = p.Parse(`<a href="tg://user?id=-1">u</a>`)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := entities[0].(*tl.MessageEntityTextURL); !ok {
		t.Errorf("expected TextURL fallback for invalid id, got %T", entities[0])
	}
}

func TestHTMLParser_Nested(t *testing.T) {
	p := NewHTMLParser()
	text, entities, err := p.Parse("<b>hello <i>world</i></b>")
	if err != nil {
		t.Fatal(err)
	}
	if text != "hello world" {
		t.Errorf("text = %q", text)
	}
	if len(entities) != 2 {
		t.Fatalf("entities = %d, want 2", len(entities))
	}
}

func TestMarkdownParser_Bold(t *testing.T) {
	p := NewMarkdownParser()
	text, entities, err := p.Parse("**hello**")
	if err != nil {
		t.Fatal(err)
	}
	if text != "hello" {
		t.Errorf("text = %q", text)
	}
	if len(entities) < 1 {
		t.Fatal("expected at least 1 entity")
	}
}

func TestMarkdownParser_Italic(t *testing.T) {
	p := NewMarkdownParser()
	text, _, err := p.Parse("*world*")
	if err != nil {
		t.Fatal(err)
	}
	if text != "world" {
		t.Errorf("text = %q", text)
	}
}

func TestParse_Dispatcher(t *testing.T) {
	text, _, err := Parse(ParseModeHTML, "<b>test</b>")
	if err != nil {
		t.Fatal(err)
	}
	if text != "test" {
		t.Errorf("text = %q", text)
	}

	text2, _, err := Parse(ParseModeDisabled, "raw text")
	if err != nil || text2 != "raw text" {
		t.Errorf("disabled mode: text=%q err=%v", text2, err)
	}
}

// --- HTML parser: hyphenated tags ---

func TestHTMLParser_TgSpoiler(t *testing.T) {
	p := NewHTMLParser()
	text, entities, err := p.Parse("<tg-spoiler>hidden</tg-spoiler>")
	if err != nil {
		t.Fatal(err)
	}
	if text != "hidden" {
		t.Errorf("text = %q", text)
	}
	if len(entities) != 1 {
		t.Fatalf("entities = %d, want 1", len(entities))
	}
	if _, ok := entities[0].(*tl.MessageEntitySpoiler); !ok {
		t.Errorf("expected MessageEntitySpoiler, got %T", entities[0])
	}
}

func TestHTMLParser_SpanSpoiler(t *testing.T) {
	p := NewHTMLParser()
	text, entities, err := p.Parse(`<span class="tg-spoiler">hidden</span>`)
	if err != nil {
		t.Fatal(err)
	}
	if text != "hidden" {
		t.Errorf("text = %q", text)
	}
	if len(entities) != 1 {
		t.Fatalf("entities = %d, want 1", len(entities))
	}
	if _, ok := entities[0].(*tl.MessageEntitySpoiler); !ok {
		t.Errorf("expected MessageEntitySpoiler, got %T", entities[0])
	}
}

func TestHTMLParser_TgEmoji(t *testing.T) {
	p := NewHTMLParser()
	text, entities, err := p.Parse(`<tg-emoji emoji-id="12345">👍</tg-emoji>`)
	if err != nil {
		t.Fatal(err)
	}
	if text != "👍" {
		t.Errorf("text = %q", text)
	}
	if len(entities) != 1 {
		t.Fatalf("entities = %d, want 1", len(entities))
	}
	emoji, ok := entities[0].(*tl.MessageEntityCustomEmoji)
	if !ok {
		t.Fatalf("expected MessageEntityCustomEmoji, got %T", entities[0])
	}
	if emoji.DocumentID != 12345 {
		t.Errorf("DocumentID = %d, want 12345", emoji.DocumentID)
	}
}

func TestHTMLParser_EmojiTag(t *testing.T) {
	p := NewHTMLParser()
	// <emoji> (without tg- prefix) also works
	text, entities, err := p.Parse(`<emoji emoji-id="42">🎉</emoji>`)
	if err != nil {
		t.Fatal(err)
	}
	if text != "🎉" {
		t.Errorf("text = %q", text)
	}
	if len(entities) != 1 {
		t.Fatalf("entities = %d, want 1", len(entities))
	}
	emoji, ok := entities[0].(*tl.MessageEntityCustomEmoji)
	if !ok {
		t.Fatalf("expected MessageEntityCustomEmoji, got %T", entities[0])
	}
	if emoji.DocumentID != 42 {
		t.Errorf("DocumentID = %d, want 42", emoji.DocumentID)
	}
}

func TestHTMLParser_EmojiWithoutID(t *testing.T) {
	p := NewHTMLParser()
	text, entities, err := p.Parse(`<tg-emoji>👍</tg-emoji>`)
	if err != nil {
		t.Fatal(err)
	}
	if text != "👍" {
		t.Errorf("text = %q", text)
	}
	if len(entities) != 0 {
		t.Errorf("entities = %d, want 0 (no emoji-id)", len(entities))
	}
}

// --- HTML parser: blockquote attributes ---

func TestHTMLParser_BlockquoteCollapsed(t *testing.T) {
	p := NewHTMLParser()
	text, entities, err := p.Parse(`<blockquote collapsed="true">text</blockquote>`)
	if err != nil {
		t.Fatal(err)
	}
	if text != "text" {
		t.Errorf("text = %q", text)
	}
	if len(entities) != 1 {
		t.Fatalf("entities = %d, want 1", len(entities))
	}
	bq, ok := entities[0].(*tl.MessageEntityBlockquote)
	if !ok {
		t.Fatalf("expected MessageEntityBlockquote, got %T", entities[0])
	}
	if !bq.Collapsed {
		t.Error("expected Collapsed=true")
	}
}

func TestHTMLParser_BlockquoteExpandable(t *testing.T) {
	p := NewHTMLParser()
	text, entities, err := p.Parse(`<blockquote expandable>text</blockquote>`)
	if err != nil {
		t.Fatal(err)
	}
	if text != "text" {
		t.Errorf("text = %q", text)
	}
	if len(entities) != 1 {
		t.Fatalf("entities = %d, want 1", len(entities))
	}
	bq, ok := entities[0].(*tl.MessageEntityBlockquote)
	if !ok {
		t.Fatalf("expected MessageEntityBlockquote, got %T", entities[0])
	}
	if !bq.Collapsed {
		t.Error("expected Collapsed=true for expandable")
	}
}

func TestHTMLParser_BlockquotePlain(t *testing.T) {
	p := NewHTMLParser()
	_, entities, err := p.Parse(`<blockquote>text</blockquote>`)
	if err != nil {
		t.Fatal(err)
	}
	bq, ok := entities[0].(*tl.MessageEntityBlockquote)
	if !ok {
		t.Fatalf("expected MessageEntityBlockquote, got %T", entities[0])
	}
	if bq.Collapsed {
		t.Error("expected Collapsed=false for plain blockquote")
	}
}

// --- HTML parser: pre with language ---

func TestHTMLParser_PreLanguage(t *testing.T) {
	p := NewHTMLParser()
	text, entities, err := p.Parse(`<pre language="go">func main() {}</pre>`)
	if err != nil {
		t.Fatal(err)
	}
	if text != "func main() {}" {
		t.Errorf("text = %q", text)
	}
	if len(entities) != 1 {
		t.Fatalf("entities = %d, want 1", len(entities))
	}
	pre, ok := entities[0].(*tl.MessageEntityPre)
	if !ok {
		t.Fatalf("expected MessageEntityPre, got %T", entities[0])
	}
	if pre.Language != "go" {
		t.Errorf("Language = %q, want go", pre.Language)
	}
}

// --- HTML parser: mailto: and URL safety ---

func TestHTMLParser_Mailto(t *testing.T) {
	p := NewHTMLParser()
	text, entities, err := p.Parse(`<a href="mailto:user@example.com">email</a>`)
	if err != nil {
		t.Fatal(err)
	}
	if text != "email" {
		t.Errorf("text = %q", text)
	}
	if len(entities) != 1 {
		t.Fatalf("entities = %d, want 1", len(entities))
	}
	if _, ok := entities[0].(*tl.MessageEntityEmail); !ok {
		t.Errorf("expected MessageEntityEmail, got %T", entities[0])
	}
}

func TestHTMLParser_DangerousURL(t *testing.T) {
	p := NewHTMLParser()
	for _, href := range []string{
		"javascript:alert(1)",
		"JaVaScRiPt:alert(1)",
		"data:text/html,<script>alert(1)</script>",
		"vbscript:msgbox(1)",
		"file:///etc/passwd",
	} {
		_, entities, err := p.Parse(`<a href="` + href + `">click</a>`)
		if err != nil {
			t.Fatal(err)
		}
		if len(entities) != 0 {
			t.Errorf("expected 0 entities for dangerous URL %q, got %d", href, len(entities))
		}
	}
}

func TestHTMLParser_EmptyHref(t *testing.T) {
	p := NewHTMLParser()
	text, entities, err := p.Parse(`<a>text</a>`)
	if err != nil {
		t.Fatal(err)
	}
	if text != "text" {
		t.Errorf("text = %q", text)
	}
	if len(entities) != 0 {
		t.Errorf("entities = %d, want 0 (empty href)", len(entities))
	}
}

// --- HTML parser: alt tag names (strong, em, ins, strike, del) ---

func TestHTMLParser_AltTagNames(t *testing.T) {
	p := NewHTMLParser()
	tests := []struct {
		input    string
		expected interface{}
	}{
		{"<strong>bold</strong>", &tl.MessageEntityBold{}},
		{"<em>italic</em>", &tl.MessageEntityItalic{}},
		{"<ins>underline</ins>", &tl.MessageEntityUnderline{}},
		{"<strike>strike</strike>", &tl.MessageEntityStrike{}},
		{"<del>delete</del>", &tl.MessageEntityStrike{}},
	}
	for _, tt := range tests {
		_, entities, err := p.Parse(tt.input)
		if err != nil {
			t.Fatalf("%q: %v", tt.input, err)
		}
		if len(entities) != 1 {
			t.Fatalf("%q: entities = %d, want 1", tt.input, len(entities))
		}
		if reflect.TypeOf(entities[0]) != reflect.TypeOf(tt.expected) {
			t.Errorf("%q: got %T, want %T", tt.input, entities[0], tt.expected)
		}
	}
}

// --- Markdown parser: links ---

func TestMarkdownParser_Link(t *testing.T) {
	p := NewMarkdownParser()
	text, entities, err := p.Parse("[click](https://example.com)")
	if err != nil {
		t.Fatal(err)
	}
	if text != "click" {
		t.Errorf("text = %q", text)
	}
	if len(entities) != 1 {
		t.Fatalf("entities = %d, want 1", len(entities))
	}
	ent, ok := entities[0].(*tl.MessageEntityTextURL)
	if !ok {
		t.Fatalf("expected MessageEntityTextURL, got %T", entities[0])
	}
	if ent.URL != "https://example.com" {
		t.Errorf("URL = %q", ent.URL)
	}
}

func TestMarkdownParser_LinkTgUser(t *testing.T) {
	p := NewMarkdownParser()
	_, entities, err := p.Parse("[user](tg://user?id=12345)")
	if err != nil {
		t.Fatal(err)
	}
	if len(entities) != 1 {
		t.Fatalf("entities = %d, want 1", len(entities))
	}
	if _, ok := entities[0].(*tl.InputMessageEntityMentionName); !ok {
		t.Errorf("expected MentionName, got %T", entities[0])
	}
}

// --- Markdown parser: underline ---

func TestMarkdownParser_Underline(t *testing.T) {
	p := NewMarkdownParser()
	text, entities, err := p.Parse("__underlined__")
	if err != nil {
		t.Fatal(err)
	}
	if text != "underlined" {
		t.Errorf("text = %q", text)
	}
	if len(entities) != 1 {
		t.Fatalf("entities = %d, want 1", len(entities))
	}
	if _, ok := entities[0].(*tl.MessageEntityUnderline); !ok {
		t.Errorf("expected MessageEntityUnderline, got %T", entities[0])
	}
}

// --- Markdown parser: strikethrough and spoiler ---

func TestMarkdownParser_Strikethrough(t *testing.T) {
	p := NewMarkdownParser()
	text, entities, err := p.Parse("~~struck~~")
	if err != nil {
		t.Fatal(err)
	}
	if text != "struck" {
		t.Errorf("text = %q", text)
	}
	if len(entities) != 1 {
		t.Fatalf("entities = %d, want 1", len(entities))
	}
	if _, ok := entities[0].(*tl.MessageEntityStrike); !ok {
		t.Errorf("expected MessageEntityStrike, got %T", entities[0])
	}
}

func TestMarkdownParser_Spoiler(t *testing.T) {
	p := NewMarkdownParser()
	text, entities, err := p.Parse("||hidden||")
	if err != nil {
		t.Fatal(err)
	}
	if text != "hidden" {
		t.Errorf("text = %q", text)
	}
	if len(entities) != 1 {
		t.Fatalf("entities = %d, want 1", len(entities))
	}
	if _, ok := entities[0].(*tl.MessageEntitySpoiler); !ok {
		t.Errorf("expected MessageEntitySpoiler, got %T", entities[0])
	}
}

// --- Markdown parser: code-first ordering ---

func TestMarkdownParser_CodeInsideBold(t *testing.T) {
	p := NewMarkdownParser()
	// **`code`** should produce bold containing a code span
	text, entities, err := p.Parse("**`code`**")
	if err != nil {
		t.Fatal(err)
	}
	if text != "code" {
		t.Errorf("text = %q", text)
	}
	// Should have both bold and code entities
	if len(entities) < 2 {
		t.Fatalf("entities = %d, want >= 2", len(entities))
	}
}

func TestMarkdownParser_BoldInsideCode(t *testing.T) {
	p := NewMarkdownParser()
	// `**not bold**` — code should suppress bold formatting
	text, entities, err := p.Parse("`**not bold**`")
	if err != nil {
		t.Fatal(err)
	}
	if text != "**not bold**" {
		t.Errorf("text = %q", text)
	}
	// Should only have a code entity, no bold
	hasBold := false
	for _, e := range entities {
		if _, ok := e.(*tl.MessageEntityBold); ok {
			hasBold = true
		}
	}
	if hasBold {
		t.Error("code should suppress bold inside it")
	}
}

// --- Markdown parser: escape handling ---

func TestMarkdownParser_EscapeAsterisk(t *testing.T) {
	p := NewMarkdownParser()
	text, entities, err := p.Parse(`\*literal\*`)
	if err != nil {
		t.Fatal(err)
	}
	if text != "*literal*" {
		t.Errorf("text = %q", text)
	}
	if len(entities) != 0 {
		t.Errorf("entities = %d, want 0 (escaped)", len(entities))
	}
}

func TestMarkdownParser_EscapeUnderscore(t *testing.T) {
	p := NewMarkdownParser()
	text, _, err := p.Parse(`\_not italic\_`)
	if err != nil {
		t.Fatal(err)
	}
	if text != "_not italic_" {
		t.Errorf("text = %q", text)
	}
}

func TestMarkdownParser_EscapeBacktick(t *testing.T) {
	p := NewMarkdownParser()
	text, _, err := p.Parse("\\`not code\\`")
	if err != nil {
		t.Fatal(err)
	}
	if text != "`not code`" {
		t.Errorf("text = %q", text)
	}
}

func TestMarkdownParser_EscapeBackslash(t *testing.T) {
	p := NewMarkdownParser()
	text, _, err := p.Parse(`\\ backslash`)
	if err != nil {
		t.Fatal(err)
	}
	if text != `\ backslash` {
		t.Errorf("text = %q", text)
	}
}

func TestMarkdownParser_EscapeMixed(t *testing.T) {
	p := NewMarkdownParser()
	text, entities, err := p.Parse(`\*escaped\* **real bold**`)
	if err != nil {
		t.Fatal(err)
	}
	if text != "*escaped* real bold" {
		t.Errorf("text = %q", text)
	}
	if len(entities) != 1 {
		t.Fatalf("entities = %d, want 1", len(entities))
	}
	if _, ok := entities[0].(*tl.MessageEntityBold); !ok {
		t.Errorf("expected MessageEntityBold, got %T", entities[0])
	}
}

func TestMarkdownParser_EscapeAngleBrackets(t *testing.T) {
	p := NewMarkdownParser()
	// \< and \> should produce literal < and > (not HTML tags)
	text, entities, err := p.Parse(`\<not a tag\>`)
	if err != nil {
		t.Fatal(err)
	}
	if text != "<not a tag>" {
		t.Errorf("text = %q", text)
	}
	if len(entities) != 0 {
		t.Errorf("entities = %d, want 0", len(entities))
	}
}

// --- Markdown parser: nested formatting (via HTML delegate) ---

func TestMarkdownParser_NestedBoldItalic(t *testing.T) {
	p := NewMarkdownParser()
	text, entities, err := p.Parse("**bold *italic* bold**")
	if err != nil {
		t.Fatal(err)
	}
	if text != "bold italic bold" {
		t.Errorf("text = %q", text)
	}
	if len(entities) < 2 {
		t.Fatalf("entities = %d, want >= 2", len(entities))
	}
}

// --- Markdown parser: complex combination ---

func TestMarkdownParser_Complex(t *testing.T) {
	p := NewMarkdownParser()
	text, entities, err := p.Parse(
		`**bold** __underline__ ~~strike~~ ||spoiler|| ` + "`code`" + ` [link](https://t.me)`,
	)
	if err != nil {
		t.Fatal(err)
	}
	if text != "bold underline strike spoiler code link" {
		t.Errorf("text = %q", text)
	}
	if len(entities) < 6 {
		t.Fatalf("entities = %d, want >= 6", len(entities))
	}
}

// --- Markdown parser: blockquotes ---

func TestMarkdownParser_Blockquote(t *testing.T) {
	p := NewMarkdownParser()
	text, entities, err := p.Parse("> quoted text")
	if err != nil {
		t.Fatal(err)
	}
	if text != "quoted text" {
		t.Errorf("text = %q", text)
	}
	if len(entities) != 1 {
		t.Fatalf("entities = %d, want 1", len(entities))
	}
	bq, ok := entities[0].(*tl.MessageEntityBlockquote)
	if !ok {
		t.Fatalf("expected MessageEntityBlockquote, got %T", entities[0])
	}
	if bq.Collapsed {
		t.Error("expected Collapsed=false for > blockquote")
	}
}

func TestMarkdownParser_BlockquoteNoSpace(t *testing.T) {
	p := NewMarkdownParser()
	text, entities, err := p.Parse(">quoted")
	if err != nil {
		t.Fatal(err)
	}
	if text != "quoted" {
		t.Errorf("text = %q", text)
	}
	if len(entities) != 1 {
		t.Fatalf("entities = %d, want 1", len(entities))
	}
	if _, ok := entities[0].(*tl.MessageEntityBlockquote); !ok {
		t.Errorf("expected MessageEntityBlockquote, got %T", entities[0])
	}
}

func TestMarkdownParser_ExpandableBlockquote(t *testing.T) {
	p := NewMarkdownParser()
	text, entities, err := p.Parse(">>> expandable quote")
	if err != nil {
		t.Fatal(err)
	}
	if text != "expandable quote" {
		t.Errorf("text = %q", text)
	}
	if len(entities) != 1 {
		t.Fatalf("entities = %d, want 1", len(entities))
	}
	bq, ok := entities[0].(*tl.MessageEntityBlockquote)
	if !ok {
		t.Fatalf("expected MessageEntityBlockquote, got %T", entities[0])
	}
	if !bq.Collapsed {
		t.Error("expected Collapsed=true for >>> blockquote")
	}
}

func TestMarkdownParser_ExpandableBlockquoteNoSpace(t *testing.T) {
	p := NewMarkdownParser()
	text, entities, err := p.Parse(">>>expandable")
	if err != nil {
		t.Fatal(err)
	}
	if text != "expandable" {
		t.Errorf("text = %q", text)
	}
	if len(entities) != 1 {
		t.Fatalf("entities = %d, want 1", len(entities))
	}
	bq, ok := entities[0].(*tl.MessageEntityBlockquote)
	if !ok {
		t.Fatalf("expected MessageEntityBlockquote, got %T", entities[0])
	}
	if !bq.Collapsed {
		t.Error("expected Collapsed=true for >>> blockquote")
	}
}

func TestMarkdownParser_BlockquoteWithFormatting(t *testing.T) {
	p := NewMarkdownParser()
	text, entities, err := p.Parse("> **bold** and *italic*")
	if err != nil {
		t.Fatal(err)
	}
	if text != "bold and italic" {
		t.Errorf("text = %q", text)
	}
	// Should have blockquote + bold + italic entities
	if len(entities) < 3 {
		t.Fatalf("entities = %d, want >= 3", len(entities))
	}
	hasBQ := false
	for _, e := range entities {
		if _, ok := e.(*tl.MessageEntityBlockquote); ok {
			hasBQ = true
		}
	}
	if !hasBQ {
		t.Error("expected a blockquote entity")
	}
}

func TestMarkdownParser_BlockquoteMultiLine(t *testing.T) {
	p := NewMarkdownParser()
	text, entities, err := p.Parse("> line one\n> line two")
	if err != nil {
		t.Fatal(err)
	}
	// Each line is a separate blockquote with a newline between them
	if text != "line one\nline two" {
		t.Errorf("text = %q", text)
	}
	if len(entities) < 2 {
		t.Fatalf("entities = %d, want >= 2", len(entities))
	}
}

func TestMarkdownParser_BlockquoteWithCode(t *testing.T) {
	p := NewMarkdownParser()
	// > inside code should be literal (code is extracted before blockquotes)
	text, entities, err := p.Parse("`> not a blockquote`")
	if err != nil {
		t.Fatal(err)
	}
	if text != "> not a blockquote" {
		t.Errorf("text = %q", text)
	}
	// Should only have a code entity, no blockquote
	for _, e := range entities {
		if _, ok := e.(*tl.MessageEntityBlockquote); ok {
			t.Error("> inside code should not create a blockquote")
		}
	}
}

func TestMarkdownParser_BlockquoteThenNormalLine(t *testing.T) {
	p := NewMarkdownParser()
	text, entities, err := p.Parse("> quoted\nnormal line")
	if err != nil {
		t.Fatal(err)
	}
	if text != "quoted\nnormal line" {
		t.Errorf("text = %q", text)
	}
	// Only the first line should be a blockquote
	if len(entities) != 1 {
		t.Fatalf("entities = %d, want 1", len(entities))
	}
	if _, ok := entities[0].(*tl.MessageEntityBlockquote); !ok {
		t.Errorf("expected MessageEntityBlockquote, got %T", entities[0])
	}
}

func TestMarkdownParser_EscapedBlockquote(t *testing.T) {
	p := NewMarkdownParser()
	// \> at start of line should NOT create a blockquote
	text, entities, err := p.Parse(`\> not a quote`)
	if err != nil {
		t.Fatal(err)
	}
	if text != "> not a quote" {
		t.Errorf("text = %q", text)
	}
	if len(entities) != 0 {
		t.Errorf("entities = %d, want 0", len(entities))
	}
}

// --- Markdown parser: escape inside code ---

func TestMarkdownParser_EscapeInsideCode(t *testing.T) {
	p := NewMarkdownParser()
	// \* inside `code` should be preserved literally, not unescaped to *
	text, entities, err := p.Parse("`\\*literal\\*`")
	if err != nil {
		t.Fatal(err)
	}
	// The backslashes should be preserved — code spans suppress all formatting
	if text != `\*literal\*` {
		t.Errorf("text = %q, want \\*literal\\*", text)
	}
	// Should only have a code entity
	if len(entities) != 1 {
		t.Fatalf("entities = %d, want 1", len(entities))
	}
	if _, ok := entities[0].(*tl.MessageEntityCode); !ok {
		t.Errorf("expected MessageEntityCode, got %T", entities[0])
	}
}

func TestMarkdownParser_EscapedBacktick(t *testing.T) {
	p := NewMarkdownParser()
	// \`text\` should be literal backticks, not a code span
	text, entities, err := p.Parse("\\`not code\\`")
	if err != nil {
		t.Fatal(err)
	}
	if text != "`not code`" {
		t.Errorf("text = %q", text)
	}
	// No entities — escaped backticks are literal
	if len(entities) != 0 {
		t.Errorf("entities = %d, want 0", len(entities))
	}
}

func TestMarkdownParser_EscapeHash(t *testing.T) {
	p := NewMarkdownParser()
	// \# should produce literal # (even though # has no special meaning in our parser)
	text, _, err := p.Parse(`\# tag`)
	if err != nil {
		t.Fatal(err)
	}
	if text != "# tag" {
		t.Errorf("text = %q, want # tag", text)
	}
}

func TestMarkdownParser_EscapedBoldInsideCode(t *testing.T) {
	p := NewMarkdownParser()
	// `\**not bold\**` — escaped asterisks inside code should NOT become bold
	text, entities, err := p.Parse("`\\**not bold\\**`")
	if err != nil {
		t.Fatal(err)
	}
	if text != `\**not bold\**` {
		t.Errorf("text = %q", text)
	}
	// Should only have code entity, no bold
	for _, e := range entities {
		if _, ok := e.(*tl.MessageEntityBold); ok {
			t.Error("escaped asterisks inside code should not become bold")
		}
	}
}
