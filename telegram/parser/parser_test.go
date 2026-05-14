package parser

import (
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
