package tlgen

import (
	"os"
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
