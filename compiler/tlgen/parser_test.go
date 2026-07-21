package tlgen

import (
	"os"
	"strings"
	"testing"
)

func TestParseMinimal(t *testing.T) {
	f, err := os.Open("testdata/minimal.tl")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	combos, err := Parse(f)
	if err != nil {
		t.Fatal(err)
	}

	if len(combos) != 7 {
		t.Fatalf("expected 7 combinators, got %d", len(combos))
	}

	obj := combos[0]
	if obj.Name != "testObj" {
		t.Fatalf("expected testObj, got %s", obj.Name)
	}
	if obj.ID != 0x12345678 {
		t.Fatalf("expected 0x12345678, got 0x%x", obj.ID)
	}
	if obj.Section != SectionTypes {
		t.Fatal("expected SectionTypes")
	}
	if obj.Type != "TestObj" {
		t.Fatalf("expected TestObj, got %s", obj.Type)
	}
	if len(obj.Args) != 1 || obj.Args[0].Name != "val" || obj.Args[0].Type != "int" {
		t.Fatal("wrong args for testObj")
	}

	flags := combos[1]
	if flags.Name != "testFlags" {
		t.Fatalf("expected testFlags, got %s", flags.Name)
	}
	if !flags.HasFlags() {
		t.Fatal("expected HasFlags true")
	}
	if len(flags.Args) != 3 {
		t.Fatalf("expected 3 args, got %d", len(flags.Args))
	}
	if flags.Args[0].Name != "flags" {
		t.Fatalf("expected first arg 'flags', got %s", flags.Args[0].Name)
	}
	if flags.Args[1].FlagBit != 0 || flags.Args[1].FlagName != "flags" {
		t.Fatal("wrong flag info for val arg")
	}

	vec := combos[2]
	if vec.Args[0].Type != "Vector<int>" {
		t.Fatalf("expected Vector<int>, got %s", vec.Args[0].Type)
	}

	fn := combos[5]
	if fn.Section != SectionFunctions {
		t.Fatal("expected SectionFunctions")
	}
	if fn.Name != "testFunc" {
		t.Fatalf("expected testFunc, got %s", fn.Name)
	}
	if fn.ID != 0x87654321 {
		t.Fatalf("expected 0x87654321, got 0x%x", fn.ID)
	}
}

func TestParseInvalidHexID(t *testing.T) {
	// Hex value too wide for uint32 (> 8 digits) triggers a ParseUint error.
	src := "badObj#123456789 val:int = BadObj;"
	_, err := Parse(strings.NewReader(src))
	if err == nil {
		t.Fatal("expected error for oversized hex id, got nil")
	}
	if !strings.Contains(err.Error(), "invalid constructor id") {
		t.Fatalf("error should mention constructor id, got: %v", err)
	}
	if !strings.Contains(err.Error(), "line 1") {
		t.Fatalf("error should include line number, got: %v", err)
	}
}

func TestParseInvalidFlagBit(t *testing.T) {
	// A flag bit that overflows int triggers an Atoi error.
	src := "badFlags#abcdef00 flags:# val:flags.9999999999999999999999?int = BadFlags;"
	_, err := Parse(strings.NewReader(src))
	if err == nil {
		t.Fatal("expected error for invalid flag bit, got nil")
	}
	if !strings.Contains(err.Error(), "invalid flag bit") {
		t.Fatalf("error should mention flag bit, got: %v", err)
	}
}

func TestParseFlagBitOutOfRange(t *testing.T) {
	src := "badFlags#abcdef00 flags:# val:flags.32?int = BadFlags;"
	_, err := Parse(strings.NewReader(src))
	if err == nil {
		t.Fatal("expected error for flag bit 32, got nil")
	}
	if !strings.Contains(err.Error(), "between 0 and 31") {
		t.Fatalf("error should include the valid flag range, got: %v", err)
	}
}

func TestParseImplicitConstructorID(t *testing.T) {
	combos, err := Parse(strings.NewReader("user id:int first_name:string last_name:string = User;"))
	if err != nil {
		t.Fatal(err)
	}
	if len(combos) != 1 {
		t.Fatalf("expected 1 combinator, got %d", len(combos))
	}
	if combos[0].ID != 0xd23c81a3 {
		t.Fatalf("expected implicit constructor ID 0xd23c81a3, got 0x%08x", combos[0].ID)
	}
}

func TestParseRejectsMalformedDeclaration(t *testing.T) {
	_, err := Parse(strings.NewReader("bad declaration"))
	if err == nil || !strings.Contains(err.Error(), "unsupported TL declaration") {
		t.Fatalf("expected unsupported declaration error, got: %v", err)
	}
}

func TestParseRejectsMalformedArgument(t *testing.T) {
	_, err := Parse(strings.NewReader("bad#abcdef00 value:int stray = Bad;"))
	if err == nil || !strings.Contains(err.Error(), "unsupported argument token") {
		t.Fatalf("expected unsupported argument error, got: %v", err)
	}
}

func TestParseRejectsUnknownSection(t *testing.T) {
	_, err := Parse(strings.NewReader("---unknown---"))
	if err == nil || !strings.Contains(err.Error(), "unknown section") {
		t.Fatalf("expected unknown section error, got: %v", err)
	}
}

func TestParseBuiltinPrimitivesSkipped(t *testing.T) {
	// Builtin primitive declarations are explicitly recognized and skipped.
	src := `int ? = Int;
long ? = Long;
string ? = String;
vector {t:Type} # [ t ] = Vector t;
vector#1cb5c415 {t:Type} # [ t ] = Vector t;
realObj#deadbeef val:int = RealObj;`
	combos, err := Parse(strings.NewReader(src))
	if err != nil {
		t.Fatalf("expected no error for builtins, got: %v", err)
	}
	if len(combos) != 1 {
		t.Fatalf("expected 1 combinator (builtins skipped), got %d", len(combos))
	}
	if combos[0].Name != "realObj" {
		t.Fatalf("expected realObj, got %s", combos[0].Name)
	}
}

func TestParseCheckedInSchemas(t *testing.T) {
	for _, path := range []string{"../source/api.tl", "../source/mtproto.tl", "../e2e.tl"} {
		t.Run(path, func(t *testing.T) {
			file, err := os.Open(path)
			if err != nil {
				t.Fatal(err)
			}
			defer file.Close()
			if _, err := Parse(file); err != nil {
				t.Fatal(err)
			}
		})
	}
}
