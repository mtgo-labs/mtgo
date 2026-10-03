package parser

import (
	"testing"

	tl "github.com/mtgo-labs/mtgo/tg"
)

func TestMarkdownV2_OfficialDelimiters(t *testing.T) {
	p := NewMarkdownParser()
	text, entities, err := p.Parse("*bold text* _italic text_ __underline__ ~strikethrough~ ||spoiler||")
	if err != nil {
		t.Fatal(err)
	}
	if text != "bold text italic text underline strikethrough spoiler" {
		t.Fatalf("text = %q", text)
	}
	want := []struct {
		kind tl.MessageEntityClass
	}{
		{&tl.MessageEntityBold{}},
		{&tl.MessageEntityItalic{}},
		{&tl.MessageEntityUnderline{}},
		{&tl.MessageEntityStrike{}},
		{&tl.MessageEntitySpoiler{}},
	}
	if len(entities) != len(want) {
		t.Fatalf("entities = %d, want %d (%v)", len(entities), len(want), entities)
	}
	for i, w := range want {
		if got := entities[i]; got.ConstructorID() != w.kind.ConstructorID() {
			t.Errorf("entity %d = %T, want %T", i, got, w.kind)
		}
	}
}

func TestMarkdownV2_DoubleDelimitersAreArtifacts(t *testing.T) {
	// **text** under the official grammar is an empty bold pair around text —
	// zero-length entities are dropped, leaving plain text.
	p := NewMarkdownParser()
	text, entities, err := p.Parse("**hello** ~~gone~~")
	if err != nil {
		t.Fatal(err)
	}
	if text != "hello gone" {
		t.Fatalf("text = %q", text)
	}
	if len(entities) != 0 {
		t.Fatalf("entities = %v, want none", entities)
	}
}

func TestMarkdownV2_CustomEmojiImageLink(t *testing.T) {
	p := NewMarkdownParser()
	text, entities, err := p.Parse("![\U0001F44D](tg://emoji?id=5368324170671202286)")
	if err != nil {
		t.Fatal(err)
	}
	if text != "\U0001F44D" {
		t.Fatalf("text = %q", text)
	}
	if len(entities) != 1 {
		t.Fatalf("entities = %d, want 1", len(entities))
	}
	ce, ok := entities[0].(*tl.MessageEntityCustomEmoji)
	if !ok {
		t.Fatalf("entity type = %T", entities[0])
	}
	// A single astral emoji is 2 UTF-16 code units.
	if ce.DocumentID != 5368324170671202286 || ce.Offset != 0 || ce.Length != 2 {
		t.Fatalf("custom emoji entity = %+v", ce)
	}
}

func TestMarkdownV2_DateTimeImageLinks(t *testing.T) {
	p := NewMarkdownParser()

	// Full format.
	text, entities, err := p.Parse("![22:45 tomorrow](tg://time?unix=1647531900&format=wDT)")
	if err != nil {
		t.Fatal(err)
	}
	if text != "22:45 tomorrow" {
		t.Fatalf("text = %q", text)
	}
	fd, ok := entities[0].(*tl.MessageEntityFormattedDate)
	if !ok {
		t.Fatalf("entity type = %T", entities[0])
	}
	if fd.Date != 1647531900 || !fd.DayOfWeek || !fd.LongDate || !fd.LongTime || fd.ShortDate || fd.ShortTime || fd.Relative {
		t.Fatalf("formatted date flags = %+v", fd)
	}

	// No format.
	_, entities, _ = p.Parse("![22:45](tg://time?unix=1647531900)")
	if fd, ok := entities[0].(*tl.MessageEntityFormattedDate); !ok || fd.Date != 1647531900 || fd.DayOfWeek || fd.Relative {
		t.Fatalf("no-format date entity = %+v (%T)", entities[0], entities[0])
	}

	// Relative format.
	_, entities, _ = p.Parse("![soon](tg://time?unix=1647531900&format=r)")
	fd, ok = entities[0].(*tl.MessageEntityFormattedDate)
	if !ok || !fd.Relative {
		t.Fatalf("relative date entity = %+v (%T)", entities[0], entities[0])
	}
}

func TestMarkdownV2_ExpandableBlockquoteMarker(t *testing.T) {
	p := NewMarkdownParser()
	// Official example: a plain blockquote immediately followed by an
	// expandable one, separated by the ** empty-bold trick.
	src := ">Block quotation\n**>The expandable block quotation\n>Hidden ||by default|| part"
	text, entities, err := p.Parse(src)
	if err != nil {
		t.Fatal(err)
	}
	if text != "Block quotation\nThe expandable block quotation\nHidden by default part" {
		t.Fatalf("text = %q", text)
	}
	var quotes []*tl.MessageEntityBlockquote
	var spoilers int
	for _, e := range entities {
		if q, ok := e.(*tl.MessageEntityBlockquote); ok {
			quotes = append(quotes, q)
		}
		if _, ok := e.(*tl.MessageEntitySpoiler); ok {
			spoilers++
		}
	}
	if len(quotes) != 2 {
		t.Fatalf("blockquotes = %d, want 2 (plain + expandable)", len(quotes))
	}
	if !quotes[1].Collapsed {
		t.Fatalf("second blockquote should be expandable: %+v", quotes[1])
	}
	if quotes[0].Collapsed {
		t.Fatalf("first blockquote should be plain: %+v", quotes[0])
	}
	if spoilers != 1 {
		t.Fatalf("spoilers = %d, want 1", spoilers)
	}
}

func TestLegacyMarkdown(t *testing.T) {
	p := NewLegacyMarkdownParser()
	text, entities, err := p.Parse("*bold text* _italic text_ [inline URL](http://www.example.com/) `code`")
	if err != nil {
		t.Fatal(err)
	}
	if text != "bold text italic text inline URL code" {
		t.Fatalf("text = %q", text)
	}
	if len(entities) != 4 {
		t.Fatalf("entities = %d, want 4 (%v)", len(entities), entities)
	}
	if _, ok := entities[0].(*tl.MessageEntityBold); !ok {
		t.Errorf("entity 0 = %T, want bold", entities[0])
	}
	if _, ok := entities[1].(*tl.MessageEntityItalic); !ok {
		t.Errorf("entity 1 = %T, want italic", entities[1])
	}
	if _, ok := entities[2].(*tl.MessageEntityTextURL); !ok {
		t.Errorf("entity 2 = %T, want text URL", entities[2])
	}
	if _, ok := entities[3].(*tl.MessageEntityCode); !ok {
		t.Errorf("entity 3 = %T, want code", entities[3])
	}
}

func TestLegacyMarkdown_NoV2OnlyEntities(t *testing.T) {
	p := NewLegacyMarkdownParser()
	// Underline, strikethrough, spoiler, and blockquotes do not exist in the
	// legacy style; the markup must come through as literal text.
	text, entities, err := p.Parse("__underline__ ~strike~ ||spoiler|| >quote")
	if err != nil {
		t.Fatal(err)
	}
	if text != "__underline__ ~strike~ ||spoiler|| >quote" {
		t.Fatalf("text = %q", text)
	}
	if len(entities) != 0 {
		t.Fatalf("entities = %v, want none", entities)
	}
}

func TestLegacyMarkdown_Escapes(t *testing.T) {
	p := NewLegacyMarkdownParser()
	text, entities, err := p.Parse(`\_not italic\_ 2\*2 \[bracket`)
	if err != nil {
		t.Fatal(err)
	}
	if text != "_not italic_ 2*2 [bracket" {
		t.Fatalf("text = %q", text)
	}
	if len(entities) != 0 {
		t.Fatalf("entities = %v, want none", entities)
	}
}

func TestHTMLParser_TgTime(t *testing.T) {
	p := NewHTMLParser()
	text, entities, err := p.Parse(`See <tg-time unix="1647531900" format="wDT">22:45 tomorrow</tg-time> now`)
	if err != nil {
		t.Fatal(err)
	}
	if text != "See 22:45 tomorrow now" {
		t.Fatalf("text = %q", text)
	}
	if len(entities) != 1 {
		t.Fatalf("entities = %d, want 1", len(entities))
	}
	fd, ok := entities[0].(*tl.MessageEntityFormattedDate)
	if !ok {
		t.Fatalf("entity type = %T", entities[0])
	}
	if fd.Offset != 4 || fd.Length != int32(len("22:45 tomorrow")) {
		t.Fatalf("offset/length = %d/%d", fd.Offset, fd.Length)
	}
	if fd.Date != 1647531900 || !fd.DayOfWeek || !fd.LongDate || !fd.LongTime {
		t.Fatalf("flags = %+v", fd)
	}
	fd.SetFlags()
	if !fd.Flags.Has(5) || !fd.Flags.Has(4) || !fd.Flags.Has(2) {
		t.Fatalf("formatted date flag bits not set: %032b", fd.Flags)
	}
}

func TestHTMLParser_TgTimeNoFormat(t *testing.T) {
	p := NewHTMLParser()
	_, entities, err := p.Parse(`<tg-time unix="1647531900">22:45</tg-time>`)
	if err != nil {
		t.Fatal(err)
	}
	if len(entities) != 1 {
		t.Fatalf("entities = %d, want 1", len(entities))
	}
	fd, ok := entities[0].(*tl.MessageEntityFormattedDate)
	if !ok || fd.Date != 1647531900 || fd.Relative || fd.DayOfWeek {
		t.Fatalf("entity = %+v (%T)", entities[0], entities[0])
	}
}

func TestHTMLParser_TgTimeInvalid(t *testing.T) {
	p := NewHTMLParser()
	cases := []string{
		`<tg-time format="t">22:45</tg-time>`,                 // missing unix
		`<tg-time unix="abc" format="t">x</tg-time>`,          // non-numeric unix
		`<tg-time unix="0">x</tg-time>`,                       // zero date
		`<tg-time unix="1647531900" format="rx">x</tg-time>`,  // bad format
		`<tg-time unix="1647531900" format="wtr">x</tg-time>`, // bad order
	}
	for _, src := range cases {
		text, entities, err := p.Parse(src)
		if err != nil {
			t.Fatalf("Parse(%q) error: %v", src, err)
		}
		if len(entities) != 0 {
			t.Errorf("Parse(%q) entities = %v, want none", src, entities)
		}
		if text == "" {
			t.Errorf("Parse(%q) produced empty text", src)
		}
	}
}

func TestHTMLParser_PreCodeLanguage(t *testing.T) {
	p := NewHTMLParser()
	text, entities, err := p.Parse("<pre><code class=\"language-python\">print('hi')</code></pre>")
	if err != nil {
		t.Fatal(err)
	}
	if text != "print('hi')" {
		t.Fatalf("text = %q", text)
	}
	if len(entities) != 1 {
		t.Fatalf("entities = %d (%v), want a single pre entity", len(entities), entities)
	}
	pre, ok := entities[0].(*tl.MessageEntityPre)
	if !ok {
		t.Fatalf("entity type = %T, want pre", entities[0])
	}
	if pre.Language != "python" {
		t.Fatalf("language = %q, want python", pre.Language)
	}
}

func TestHTMLParser_PlainCodeHasNoLanguage(t *testing.T) {
	p := NewHTMLParser()
	_, entities, err := p.Parse("a <code class=\"language-go\">x</code> b")
	if err != nil {
		t.Fatal(err)
	}
	// A language on a standalone code tag is not part of the specification;
	// the code entity carries no language either way.
	if len(entities) != 1 {
		t.Fatalf("entities = %d, want 1", len(entities))
	}
	if _, ok := entities[0].(*tl.MessageEntityCode); !ok {
		t.Fatalf("entity type = %T, want code", entities[0])
	}
}

func TestHTMLParser_NumericEntities(t *testing.T) {
	p := NewHTMLParser()
	text, _, err := p.Parse("&#60;b&#62;not bold&#60;/b&#62; &#x27;quoted&#x27; &amp;#60;stays&#60;")
	if err != nil {
		t.Fatal(err)
	}
	if text != "<b>not bold</b> 'quoted' &#60;stays<" {
		t.Fatalf("text = %q", text)
	}
}

func TestParseDateTimeFormat(t *testing.T) {
	cases := []struct {
		in                           string
		rel, st, lt, sd, ld, dow, ok bool
	}{
		{"", false, false, false, false, false, false, true},
		{"r", true, false, false, false, false, false, true},
		{"t", false, true, false, false, false, false, true},
		{"T", false, false, true, false, false, false, true},
		{"d", false, false, false, true, false, false, true},
		{"D", false, false, false, false, true, false, true},
		{"w", false, false, false, false, false, true, true},
		{"wdt", false, true, false, true, false, true, true},
		{"wDT", false, false, true, false, true, true, true},
		{"dt", false, true, false, true, false, false, true},
		{"x", false, false, false, false, false, false, false},
		{"rw", false, false, false, false, false, false, false},
		{"td", false, false, false, false, false, false, false},
	}
	for _, c := range cases {
		rel, st, lt, sd, ld, dow, ok := parseDateTimeFormat(c.in)
		if ok != c.ok || rel != c.rel || st != c.st || lt != c.lt || sd != c.sd || ld != c.ld || dow != c.dow {
			t.Errorf("parseDateTimeFormat(%q) = (r:%v t:%v T:%v d:%v D:%v w:%v ok:%v), want (r:%v t:%v T:%v d:%v D:%v w:%v ok:%v)",
				c.in, rel, st, lt, sd, ld, dow, ok, c.rel, c.st, c.lt, c.sd, c.ld, c.dow, c.ok)
		}
	}
}
