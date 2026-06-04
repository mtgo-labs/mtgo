package tlgen

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGoType(t *testing.T) {
	tests := []struct {
		tlType string
		want   string
	}{
		{"int", "int32"},
		{"long", "int64"},
		{"int128", "[16]byte"},
		{"int256", "[32]byte"},
		{"double", "float64"},
		{"string", "string"},
		{"bytes", "[]byte"},
		{"Bool", "bool"},
		{"true", "bool"},
		{"Vector<int>", "[]int32"},
		{"Vector<long>", "[]int64"},
		{"Vector<string>", "[]string"},
		{"vector<future_salt>", "[]*FutureSalt"},
		{"Type", "TLObject"},
	}

	for _, tt := range tests {
		got := goType(tt.tlType, "types", nil, map[string][]Combinator{"FutureSalt": {{QualName: "future_salt"}}})
		if got != tt.want {
			t.Errorf("goType(%q) = %q, want %q", tt.tlType, got, tt.want)
		}
	}
}

func TestSnakeToPascal(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"input_peer_user", "InputPeerUser"},
		{"user", "User"},
		{"messages.getMessages", "MessagesGetMessages"},
		{"auth.sendCode", "AuthSendCode"},
		{"bad_msg_notification", "BadMsgNotification"},
		{"inputBotInlineMessageID64", "InputBotInlineMessageID64"},
		{"invokeAfterMsgs", "InvokeAfterMsgs"},
		{"p_q_inner_data", "PQInnerData"},
		{"rpc_result", "RPCResult"},
		{"storage.fileJpeg", "StorageFileJPEG"},
	}

	for _, tt := range tests {
		got := SnakeToPascal(tt.input)
		if got != tt.want {
			t.Errorf("SnakeToPascal(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestCamelToSnake(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"InputPeerUser", "input_peer_user"},
		{"User", "user"},
		{"SendCode", "send_code"},
		{"BadMsgNotification", "bad_msg_notification"},
	}

	for _, tt := range tests {
		got := CamelToSnake(tt.input)
		if got != tt.want {
			t.Errorf("CamelToSnake(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestReadExpr(t *testing.T) {
	tests := []struct {
		arg    Arg
		flags  string
		want   string
		isFunc bool
	}{
		{Arg{Name: "val", Type: "int"}, "", "r.ReadInt32()", false},
		{Arg{Name: "id", Type: "long"}, "", "r.ReadInt64()", false},
		{Arg{Name: "data", Type: "string"}, "", "r.ReadString()", false},
		{Arg{Name: "b", Type: "bytes"}, "", "r.ReadBytes()", false},
		{Arg{Name: "x", Type: "double"}, "", "r.ReadFloat64()", false},
		{Arg{Name: "n", Type: "int128"}, "", "r.ReadInt128()", false},
		{Arg{Name: "n", Type: "int256"}, "", "r.ReadInt256()", false},
		{Arg{Name: "ok", Type: "Bool"}, "", "r.ReadBool()", false},
		{Arg{Name: "ok", Type: "true"}, "", "r.ReadBool()", false},
		{Arg{Name: "items", Type: "Vector<int>"}, "", "r.ReadVectorInt()", false},
		{Arg{Name: "ids", Type: "Vector<long>"}, "", "r.ReadVectorLong()", false},
		{Arg{Name: "strs", Type: "Vector<string>"}, "", "r.ReadVectorString()", false},
		{Arg{Name: "access_hash", Type: "long", FlagBit: 0, FlagName: "flags"}, "flags", "r.ReadInt64() if flags&(1<<0) != 0", false},
		{Arg{Name: "photo", Type: "Photo", FlagBit: 5, FlagName: "flags"}, "flags", "ReadTLObject(r) if flags&(1<<5) != 0", false},
	}

	for _, tt := range tests {
		got := readFuncName(tt.arg.Type, "types", nil)
		want := strings.TrimSuffix(tt.want, " if flags&(1<<0) != 0")
		want = strings.TrimSuffix(want, " if flags&(1<<5) != 0")
		if got != want {
			t.Errorf("readFuncName(%q) = %q, want %q", tt.arg.Type, got, want)
		}
	}
}

func TestWriteExpr(t *testing.T) {
	tests := []struct {
		arg    Arg
		goType string
		want   string
	}{
		{Arg{Name: "val", Type: "int"}, "int32", "WriteInt(b, uint32(v.Val))"},
		{Arg{Name: "id", Type: "long"}, "int64", "WriteLong(b, v.ID)"},
		{Arg{Name: "data", Type: "string"}, "string", "WriteString(b, v.Data)"},
		{Arg{Name: "b", Type: "bytes"}, "[]byte", "WriteBytes(b, v.B)"},
		{Arg{Name: "x", Type: "double"}, "float64", "WriteDouble(b, v.X)"},
		{Arg{Name: "n", Type: "int128"}, "[16]byte", "WriteInt128(b, v.N)"},
		{Arg{Name: "n", Type: "int256"}, "[32]byte", "WriteInt256(b, v.N)"},
		{Arg{Name: "ok", Type: "Bool"}, "bool", "WriteBool(b, v.Ok)"},
		{Arg{Name: "ok", Type: "true"}, "bool", ""},
		{Arg{Name: "items", Type: "Vector<int>"}, "[]int32", "WriteVectorInt(b, v.Items)"},
	}

	for _, tt := range tests {
		got := writeExpr(tt.arg, tt.goType, "v")
		if got != tt.want {
			t.Errorf("writeExpr(%+v, %q) = %q, want %q", tt.arg, tt.goType, got, tt.want)
		}
	}
}

func TestWriteExprBareVectorConstructor(t *testing.T) {
	arg := Arg{Name: "salts", Type: "vector<future_salt>"}
	typeToConstructor := map[string][]Combinator{
		"FutureSalt": {{
			QualName: "future_salt",
			Type:     "FutureSalt",
			Args: []Arg{
				{Name: "valid_since", Type: "int", FlagBit: -1},
				{Name: "valid_until", Type: "int", FlagBit: -1},
				{Name: "salt", Type: "long", FlagBit: -1},
			},
		}},
	}

	got := writeExpr(arg, "[]*FutureSalt", "v", typeToConstructor)
	if strings.Contains(got, "0x1cb5c415") {
		t.Fatalf("bare vector write should not include vector constructor: %s", got)
	}
	if strings.Contains(got, "EncodeTLObject") {
		t.Fatalf("bare vector element write should not use boxed object encoding: %s", got)
	}
	for _, want := range []string{"WriteInt(b, uint32(len(v.Salts)))", "_item.ValidSince", "_item.ValidUntil", "_item.Salt"} {
		if !strings.Contains(got, want) {
			t.Fatalf("writeExpr missing %q in %s", want, got)
		}
	}
}

func TestBareVectorBoxedElements(t *testing.T) {
	arg := Arg{Name: "ips", Type: "vector<IpPort>"}

	write := writeExpr(arg, "[]IpPortClass", "v", nil)
	if strings.Contains(write, "0x1cb5c415") {
		t.Fatalf("bare vector write should not include vector constructor: %s", write)
	}
	if !strings.Contains(write, "EncodeTLObject") {
		t.Fatalf("boxed vector elements should use TL object encoding: %s", write)
	}

	read := buildReadExpr(arg, "[]IpPortClass", nil, nil)
	if strings.Contains(read, "_vhdr") {
		t.Fatalf("bare vector read should not include vector header: %s", read)
	}
	if !strings.Contains(read, "ReadTLObject(r)") || !strings.Contains(read, ".(IpPortClass)") {
		t.Fatalf("boxed vector elements should use TL object decoding: %s", read)
	}
	if strings.Index(read, "if _errIps != nil") > strings.Index(read, "_objIps.(IpPortClass)") {
		t.Fatalf("read error should be checked before type assertion: %s", read)
	}
}

func TestBuildReadExprBareVectorConstructor(t *testing.T) {
	arg := Arg{Name: "salts", Type: "vector<future_salt>"}
	typeToConstructor := map[string][]Combinator{
		"FutureSalt": {{QualName: "future_salt", Type: "FutureSalt"}},
	}

	got := buildReadExpr(arg, "[]*FutureSalt", nil, typeToConstructor)
	if strings.Contains(got, "_vhdr") || strings.Contains(got, "ReadTLObject") {
		t.Fatalf("bare vector read should not use boxed vector/object decoding: %s", got)
	}
	for _, want := range []string{"_cntSalts", "checkVectorCount", "DecodeFutureSalt(r)"} {
		if !strings.Contains(got, want) {
			t.Fatalf("buildReadExpr missing %q in %s", want, got)
		}
	}
}

func TestGenerateTypes(t *testing.T) {
	combos := []Combinator{
		{
			Section:  SectionTypes,
			QualName: "testObj",
			Name:     "TestObj",
			ID:       0x12345678,
			Args: []Arg{
				{Name: "val", Type: "int", FlagBit: -1},
			},
			QualType: "TestObj",
			Type:     "TestObj",
		},
		{
			Section:  SectionTypes,
			QualName: "testFlags",
			Name:     "TestFlags",
			ID:       0xABCDEF00,
			Args: []Arg{
				{Name: "flags", Type: "#", FlagBit: -1},
				{Name: "val", Type: "true", FlagBit: 0, FlagName: "flags"},
				{Name: "opt_bool", Type: "Bool", FlagBit: 1, FlagName: "flags"},
				{Name: "opt_string", Type: "string", FlagBit: 2, FlagName: "flags"},
				{Name: "opt_long", Type: "long", FlagBit: 3, FlagName: "flags"},
				{Name: "other", Type: "string", FlagBit: -1},
			},
			QualType: "TestFlags",
			Type:     "TestFlags",
		},
	}

	tmpDir := t.TempDir()
	err := GenerateGroupedTypes(tmpDir, combos, 224)
	if err != nil {
		t.Fatal(err)
	}

	files, _ := filepath.Glob(filepath.Join(tmpDir, "*_types_gen.go"))
	if len(files) == 0 {
		t.Fatal("no type files generated")
	}
	var content string
	for _, f := range files {
		data, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		content += string(data)
	}

	if !strings.Contains(content, "type TestObj struct") {
		t.Fatal("missing struct definition")
	}
	if !strings.Contains(content, "Val int32") {
		t.Fatal("missing Val field")
	}
	if !strings.Contains(content, "func (v *TestObj) ConstructorID() uint32") {
		t.Fatal("missing ConstructorID method")
	}
	if !strings.Contains(content, "0x12345678") {
		t.Fatal("missing constructor ID")
	}

	if !strings.Contains(content, "type TestFlags struct") {
		t.Fatal("missing TestFlags struct")
	}
	if !strings.Contains(content, "Flags") || !strings.Contains(content, "uint32") {
		t.Fatal("missing Flags field")
	}
	if !strings.Contains(content, "Val") || !strings.Contains(content, "bool") {
		t.Fatal("missing Val field")
	}
	if !strings.Contains(content, "OptBool") {
		t.Fatal("missing optional Bool field")
	}
	if !strings.Contains(content, "OptBool") || !strings.Contains(content, "bool") {
		t.Fatal("optional Bool field should be direct bool")
	}
	if !strings.Contains(content, "if v.OptBool {") {
		t.Fatal("optional Bool flag should be set from direct bool value")
	}
	if !strings.Contains(content, "v.OptBool = _rOptBool") {
		t.Fatal("optional Bool field should decode into direct bool value")
	}
	if !strings.Contains(content, "OptString") || !strings.Contains(content, "string") {
		t.Fatal("optional string field should be direct string")
	}
	if !strings.Contains(content, "if v.OptString != \"\" {") {
		t.Fatal("optional string flag should be set from direct string value")
	}
	if !strings.Contains(content, "v.OptString = _rOptString") {
		t.Fatal("optional string field should decode into direct string value")
	}
	if !strings.Contains(content, "OptLong") || !strings.Contains(content, "int64") {
		t.Fatal("optional long field should be direct int64")
	}
	if !strings.Contains(content, "if v.OptLong != 0 {") {
		t.Fatal("optional long flag should be set from direct int64 value")
	}
	if !strings.Contains(content, "v.OptLong = _rOptLong") {
		t.Fatal("optional long field should decode into direct int64 value")
	}
	if !strings.Contains(content, "init()") {
		t.Fatal("missing init() for registry")
	}
}

func TestGenerateFunctions(t *testing.T) {
	combos := []Combinator{
		{
			Section:  SectionFunctions,
			QualName: "testFunc",
			Name:     "TestFunc",
			ID:       0x87654321,
			Args: []Arg{
				{Name: "val", Type: "string", FlagBit: -1},
			},
			QualType: "TestObj",
			Type:     "TestObj",
		},
		{
			Section:  SectionFunctions,
			QualName: "testFuncFlags",
			Name:     "TestFuncFlags",
			ID:       0x00ABCDEF,
			Args: []Arg{
				{Name: "flags", Type: "#", FlagBit: -1},
				{Name: "opt_val", Type: "string", FlagBit: 5, FlagName: "flags"},
			},
			QualType: "TestObj",
			Type:     "TestObj",
		},
	}

	tmpDir := t.TempDir()
	err := GenerateGroupedFunctions(tmpDir, combos, 224)
	if err != nil {
		t.Fatal(err)
	}

	files, _ := filepath.Glob(filepath.Join(tmpDir, "*_methods_gen.go"))
	if len(files) == 0 {
		t.Fatal("no method files generated")
	}
	var content string
	for _, f := range files {
		data, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		content += string(data)
	}
	if !strings.Contains(content, "type TestFuncRequest struct") {
		t.Fatal("missing struct")
	}
	if !strings.Contains(content, "0x87654321") {
		t.Fatal("missing ID")
	}
	if !strings.Contains(content, "OptVal") {
		t.Fatal("missing optional string field")
	}
}

func TestGenerateBases(t *testing.T) {
	combos := []Combinator{
		{
			Section:  SectionTypes,
			QualName: "user",
			Name:     "User",
			ID:       0x11111111,
			Type:     "User",
		},
		{
			Section:  SectionTypes,
			QualName: "userEmpty",
			Name:     "UserEmpty",
			ID:       0x22222222,
			Type:     "User",
		},
		{
			Section:  SectionTypes,
			QualName: "chat",
			Name:     "Chat",
			ID:       0x33333333,
			Type:     "Chat",
		},
		{
			Section:  SectionTypes,
			QualName: "chatEmpty",
			Name:     "ChatEmpty",
			ID:       0x44444444,
			Type:     "Chat",
		},
	}

	tmpDir := t.TempDir()
	err := GenerateInterfaces(tmpDir, combos)
	if err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(filepath.Join(tmpDir, "interfaces_gen.go"))
	if err != nil {
		t.Fatal(err)
	}
	content := string(data)
	if !strings.Contains(content, "type UserClass interface") {
		t.Fatal("missing UserClass interface")
	}
	if !strings.Contains(content, "isUser()") {
		t.Fatal("missing User marker method")
	}
	if !strings.Contains(content, "type ChatClass interface") {
		t.Fatal("missing ChatClass interface")
	}
}
