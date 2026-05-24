package tlgen

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFlagArgs(t *testing.T) {
	c := &Combinator{
		Args: []Arg{
			{Name: "flags", Type: "#", FlagBit: -1},
			{Name: "val", Type: "true", FlagBit: 0, FlagName: "flags"},
			{Name: "name", Type: "string", FlagBit: 5, FlagName: "flags"},
			{Name: "other", Type: "string", FlagBit: -1},
		},
	}
	flags := c.FlagArgs()
	if len(flags) != 2 {
		t.Fatalf("FlagArgs() returned %d args, want 2", len(flags))
	}
	if flags[0].Name != "val" {
		t.Errorf("flags[0].Name = %q, want %q", flags[0].Name, "val")
	}
	if flags[1].Name != "name" {
		t.Errorf("flags[1].Name = %q, want %q", flags[1].Name, "name")
	}
}

func TestFlagArgs_Empty(t *testing.T) {
	c := &Combinator{
		Args: []Arg{
			{Name: "flags", Type: "#", FlagBit: -1},
			{Name: "val", Type: "string", FlagBit: -1},
		},
	}
	flags := c.FlagArgs()
	if len(flags) != 0 {
		t.Fatalf("FlagArgs() returned %d args, want 0", len(flags))
	}
}

func TestNonFlagArgs(t *testing.T) {
	c := &Combinator{
		Args: []Arg{
			{Name: "flags", Type: "#", FlagBit: -1},
			{Name: "val", Type: "true", FlagBit: 0, FlagName: "flags"},
			{Name: "other", Type: "string", FlagBit: -1},
			{Name: "data", Type: "long", FlagBit: -1},
		},
	}
	args := c.NonFlagArgs()
	if len(args) != 2 {
		t.Fatalf("NonFlagArgs() returned %d args, want 2", len(args))
	}
	if args[0].Name != "other" {
		t.Errorf("args[0].Name = %q, want %q", args[0].Name, "other")
	}
	if args[1].Name != "data" {
		t.Errorf("args[1].Name = %q, want %q", args[1].Name, "data")
	}
}

func TestNonFlagArgs_AllFlags(t *testing.T) {
	c := &Combinator{
		Args: []Arg{
			{Name: "flags", Type: "#", FlagBit: -1},
			{Name: "val", Type: "true", FlagBit: 0, FlagName: "flags"},
		},
	}
	args := c.NonFlagArgs()
	if len(args) != 0 {
		t.Fatalf("NonFlagArgs() returned %d args, want 0", len(args))
	}
}

func TestTypeAssertExpr_Slice(t *testing.T) {
	got := typeAssertExpr("obj", "[]int32")
	if got != "[]int32(obj)" {
		t.Errorf("typeAssertExpr slice = %q, want %q", got, "[]int32(obj)")
	}
}

func TestTypeAssertExpr_PtrClass(t *testing.T) {
	got := typeAssertExpr("obj", "*UserClass")
	if got != "obj.(UserClass)" {
		t.Errorf("typeAssertExpr ptr class = %q, want %q", got, "obj.(UserClass)")
	}
}

func TestTypeAssertExpr_PtrConcrete(t *testing.T) {
	got := typeAssertExpr("obj", "*User")
	if got != "obj.(*User)" {
		t.Errorf("typeAssertExpr ptr = %q, want %q", got, "obj.(*User)")
	}
}

func TestTypeAssertExpr_Class(t *testing.T) {
	got := typeAssertExpr("obj", "UserClass")
	if got != "obj.(UserClass)" {
		t.Errorf("typeAssertExpr class = %q, want %q", got, "obj.(UserClass)")
	}
}

func TestTypeAssertExpr_PlainType(t *testing.T) {
	got := typeAssertExpr("obj", "User")
	if got != "obj.(User)" {
		t.Errorf("typeAssertExpr plain = %q, want %q", got, "obj.(User)")
	}
}

func TestPrefixE2EType(t *testing.T) {
	tests := []struct {
		input, want string
	}{
		{"Fields", "tg.Fields"},
		{"string", "string"},
		{"int32", "int32"},
		{"*UserClass", "*UserClass"},
	}
	for _, tt := range tests {
		got := prefixE2EType(tt.input)
		if got != tt.want {
			t.Errorf("prefixE2EType(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestDomainTitle(t *testing.T) {
	tests := []struct {
		input, want string
	}{
		{"", "Core"},
		{"auth", "Auth"},
		{"messages", "Messages"},
		{"core", "Core"},
	}
	for _, tt := range tests {
		got := domainTitle(tt.input)
		if got != tt.want {
			t.Errorf("domainTitle(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestDomainFileOrder(t *testing.T) {
	domains := map[string]bool{
		"core":     true,
		"auth":     true,
		"messages": true,
		"peer":     true,
		"unknown":  true,
	}
	order := domainFileOrder(domains)
	expectedPrefix := []string{"auth", "messages", "peer"}
	for i, exp := range expectedPrefix {
		if order[i] != exp {
			t.Errorf("order[%d] = %q, want %q", i, order[i], exp)
		}
	}
	last := order[len(order)-1]
	if last != "unknown" && last != "core" {
		t.Errorf("unexpected last element %q", last)
	}
	found := false
	for _, d := range order {
		if d == "unknown" {
			found = true
		}
	}
	if !found {
		t.Error("unknown domain not in order")
	}
}

func TestClassifyTypeDomain(t *testing.T) {
	tests := []struct {
		name   string
		combos []Combinator
		want   string
	}{
		{"user", nil, "peer"},
		{"authSentCode", nil, "auth"},
		{"messageEmpty", nil, "messages"},
		{"documentAttribute", nil, "media"},
		{"updateShort", nil, "updates"},
		{"accountPrivacyRules", nil, "account"},
		{"botInfo", nil, "bots"},
		{"paymentCharge", nil, "payments"},
		{"phoneCall", nil, "phone"},
		{"storyItem", nil, "stories"},
		{"pageBlock", nil, "pages"},
		{"poll", nil, "polls"},
		{"chatlist", nil, "peer"},
		{"encryptedMessage", nil, "messages"},
		{"geoPoint", nil, "geo"},
		{"config", nil, "core"},
		{"smsJob", nil, "smsjobs"},
		{"aiComposeResult", nil, "aicompose"},
		{"randomThing", nil, "core"},
	}
	for _, tt := range tests {
		got := classifyTypeDomain(tt.name, tt.combos)
		if got != tt.want {
			t.Errorf("classifyTypeDomain(%q) = %q, want %q", tt.name, got, tt.want)
		}
	}
}

func TestClassifyTypeDomain_NamespaceSingle(t *testing.T) {
	combos := []Combinator{
		{Namespace: "messages"},
		{Namespace: "messages"},
	}
	got := classifyTypeDomain("anything", combos)
	if got != "messages" {
		t.Errorf("classifyTypeDomain single ns = %q, want %q", got, "messages")
	}
}

func TestClassifyTypeDomain_NamespaceMultiple(t *testing.T) {
	combos := []Combinator{
		{Namespace: "auth"},
		{Namespace: "auth"},
		{Namespace: "messages"},
	}
	got := classifyTypeDomain("anything", combos)
	if got != "auth" {
		t.Errorf("classifyTypeDomain multi ns = %q, want %q", got, "auth")
	}
}

func TestClassifyFuncDomain(t *testing.T) {
	tests := []struct {
		c    Combinator
		want string
	}{
		{Combinator{Namespace: "auth"}, "auth"},
		{Combinator{Namespace: "messages"}, "messages"},
		{Combinator{Namespace: ""}, "core"},
	}
	for _, tt := range tests {
		got := classifyFuncDomain(tt.c)
		if got != tt.want {
			t.Errorf("classifyFuncDomain(%+v) = %q, want %q", tt.c, got, tt.want)
		}
	}
}

func TestGenerateEnums(t *testing.T) {
	combos := []Combinator{
		{Section: SectionTypes, QualName: "boolTrue", Name: "BoolTrue", ID: 0x997275b5, Type: "Bool"},
		{Section: SectionTypes, QualName: "boolFalse", Name: "BoolFalse", ID: 0xbc799737, Type: "Bool"},
		{Section: SectionTypes, QualName: "null", Name: "Null", ID: 0x56730bcc, Type: "Null"},
	}
	tmpDir := t.TempDir()
	if err := GenerateEnums(tmpDir, combos); err != nil {
		t.Fatal(err)
	}
	data, err := readFile(tmpDir, "enums_gen.go")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(data, "BoolTrueTypeID") {
		t.Error("missing BoolTrueTypeID")
	}
	if !strings.Contains(data, "BoolFalseTypeID") {
		t.Error("missing BoolFalseTypeID")
	}
	if !strings.Contains(data, "0x997275b5") {
		t.Error("missing boolTrue ID")
	}
	if !strings.Contains(data, "EncodeBool") {
		t.Error("missing EncodeBool")
	}
	if !strings.Contains(data, "NullTypeID") {
		t.Error("missing NullTypeID")
	}
}

func TestGenerateCodecHelpers(t *testing.T) {
	combos := []Combinator{
		{Section: SectionTypes, QualName: "testObj", Name: "TestObj", ID: 0x12345678, Type: "TestObj",
			Args: []Arg{{Name: "val", Type: "int", FlagBit: -1}}},
	}
	tmpDir := t.TempDir()
	if err := GenerateCodecHelpers(tmpDir, combos); err != nil {
		t.Fatal(err)
	}
	data, err := readFile(tmpDir, "codec_gen.go")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(data, "RegisterTypes") {
		t.Error("missing RegisterTypes")
	}
	if !strings.Contains(data, "DecodeTestObj") {
		t.Error("missing DecodeTestObj in registry")
	}
	if !strings.Contains(data, "EncodeTLObject") {
		t.Error("missing EncodeTLObject")
	}
}

func TestGenerateGroupedConstructors(t *testing.T) {
	combos := []Combinator{
		{Section: SectionTypes, QualName: "user", Name: "User", ID: 0x11111111, Type: "User"},
		{Section: SectionTypes, QualName: "userEmpty", Name: "UserEmpty", ID: 0x22222222, Type: "User"},
		{Section: SectionTypes, QualName: "chat", Name: "Chat", ID: 0x33333333, Type: "Chat"},
	}
	tmpDir := t.TempDir()
	if err := GenerateGroupedConstructors(tmpDir, combos); err != nil {
		t.Fatal(err)
	}
	data, err := readFile(tmpDir, "tl_constructors_gen.go")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(data, "ConstructorMap") {
		t.Error("missing ConstructorMap")
	}
	if !strings.Contains(data, "0x11111111") {
		t.Error("missing user constructor ID")
	}
	if !strings.Contains(data, "0x22222222") {
		t.Error("missing userEmpty constructor ID")
	}
}

func TestGenerateNamesMap(t *testing.T) {
	combos := []Combinator{
		{QualName: "messages.sendMessage", ID: 0x12345678},
		{QualName: "auth.sendCode", ID: 0xABCDEF00},
	}
	tmpDir := t.TempDir()
	if err := GenerateNamesMap(tmpDir, combos); err != nil {
		t.Fatal(err)
	}
	data, err := readFile(tmpDir, "tl_names_gen.go")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(data, "NamesMap") {
		t.Error("missing NamesMap")
	}
	if !strings.Contains(data, "messages.sendMessage") {
		t.Error("missing messages.sendMessage")
	}
}

func TestGenerateFunctionsMap(t *testing.T) {
	combos := []Combinator{
		{
			Section:  SectionFunctions,
			QualName: "testFunc",
			Name:     "TestFunc",
			ID:       0x87654321,
			Args:     []Arg{{Name: "val", Type: "string", FlagBit: -1}},
			Type:     "TestObj",
		},
	}
	tmpDir := t.TempDir()
	if err := GenerateFunctionsMap(tmpDir, combos); err != nil {
		t.Fatal(err)
	}
	data, err := readFile(tmpDir, "tl_functions_gen.go")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(data, "FunctionsMap") {
		t.Error("missing FunctionsMap")
	}
	if !strings.Contains(data, "TestFuncRequest") {
		t.Error("missing TestFuncRequest")
	}
}

func TestE2EGeneration(t *testing.T) {
	combos := []Combinator{
		{
			Section:  SectionTypes,
			QualName: "encryptedMessage",
			Name:     "EncryptedMessage",
			ID:       0xAAAAAAAA,
			Args: []Arg{
				{Name: "flags", Type: "#", FlagBit: -1},
				{Name: "data", Type: "string", FlagBit: -1},
				{Name: "opt_val", Type: "true", FlagBit: 0, FlagName: "flags"},
			},
			Type: "EncryptedMessage",
		},
		{
			Section:  SectionFunctions,
			QualName: "messages.sendEncrypted",
			Name:     "MessagesSendEncrypted",
			ID:       0xBBBBBBBB,
			Args:     []Arg{{Name: "peer", Type: "InputEncryptedChat", FlagBit: -1}},
			Type:     "Bool",
		},
	}
	tmpDir := t.TempDir()
	if err := GenerateE2EPackage(tmpDir, combos, 224); err != nil {
		t.Fatal(err)
	}

	e2eGo, err := readFile(tmpDir, "e2e.go")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(e2eGo, "Registry") {
		t.Error("missing Registry in e2e.go")
	}
	if !strings.Contains(e2eGo, "ReadE2ETLObject") {
		t.Error("missing ReadE2ETLObject in e2e.go")
	}

	typesFile, err := readFileGlob(tmpDir, "*_types_gen.go")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(typesFile, "EncryptedMessage") {
		t.Error("missing EncryptedMessage struct")
	}
	if !strings.Contains(typesFile, "tg.Fields") {
		t.Error("missing tg.Fields in e2e types")
	}
	if !strings.Contains(typesFile, "tg.WriteInt") {
		t.Error("missing tg.WriteInt prefix in e2e encode")
	}

	constructorsFile, err := readFile(tmpDir, "tl_constructors_gen.go")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(constructorsFile, "ConstructorMap") {
		t.Error("missing ConstructorMap in e2e constructors")
	}

	funcsFile, err := readFileGlob(tmpDir, "*_methods_gen.go")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(funcsFile, "MessagesSendEncryptedRequest") {
		t.Error("missing request struct in e2e methods")
	}

	funcsMap, err := readFile(tmpDir, "tl_functions_gen.go")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(funcsMap, "FunctionsMap") {
		t.Error("missing FunctionsMap in e2e functions map")
	}

	namesMap, err := readFile(tmpDir, "tl_names_gen.go")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(namesMap, "encryptedMessage") {
		t.Error("missing encryptedMessage in e2e names map")
	}

	clientFile, err := readFile(tmpDir, "tl_client_gen.go")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(clientFile, "RPCClient") {
		t.Error("missing RPCClient in e2e client")
	}

	docFile, err := readFile(tmpDir, "doc.go")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(docFile, "package e2e") {
		t.Error("missing package e2e")
	}
}

func TestE2EGeneration_NoFunctions(t *testing.T) {
	combos := []Combinator{
		{
			Section:  SectionTypes,
			QualName: "encryptedMessage",
			Name:     "EncryptedMessage",
			ID:       0xAAAAAAAA,
			Type:     "EncryptedMessage",
		},
	}
	tmpDir := t.TempDir()
	if err := GenerateE2EPackage(tmpDir, combos, 224); err != nil {
		t.Fatal(err)
	}
}

func TestE2EPrefixWriteCode(t *testing.T) {
	tests := []struct {
		input, want string
	}{
		{"WriteInt(b, 0x1)", "tg.WriteInt(b, 0x1)"},
		{"WriteString(b, v.Data)", "tg.WriteString(b, v.Data)"},
		{"WriteVectorInt(b, v.Items)", "tg.WriteVectorInt(b, v.Items)"},
		{"EncodeTLObject(b, v.Obj)", "tg.EncodeTLObject(b, v.Obj)"},
		{"something else", "something else"},
	}
	for _, tt := range tests {
		got := e2ePrefixWriteCode(tt.input)
		if got != tt.want {
			t.Errorf("e2ePrefixWriteCode(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestE2EPrefixReadCode(t *testing.T) {
	tests := []struct {
		input, want string
	}{
		{"ReadTLObject(r)", "ReadE2ETLObject(r)"},
		{"checkVectorCount(10)", "tg.CheckVectorCount(10)"},
		{"Fields(5)", "tg.Fields(5)"},
		{"something else", "something else"},
	}
	for _, tt := range tests {
		got := e2ePrefixReadCode(tt.input)
		if got != tt.want {
			t.Errorf("e2ePrefixReadCode(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestIsDirectOptionalScalar(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"bool", true},
		{"int32", true},
		{"int64", true},
		{"float64", true},
		{"string", true},
		{"uint32", true},
		{"[]byte", false},
		{"[16]byte", false},
		{"*int32", false},
		{"TLObject", false},
	}
	for _, tt := range tests {
		got := isDirectOptionalScalar(tt.input)
		if got != tt.want {
			t.Errorf("isDirectOptionalScalar(%q) = %v, want %v", tt.input, got, tt.want)
		}
	}
}

func TestStripNamespace(t *testing.T) {
	tests := []struct {
		input, want string
	}{
		{"storage.fileJpeg", "fileJpeg"},
		{"messages.sendMessage", "sendMessage"},
		{"user", "user"},
		{"Vector<storage.fileJpeg>", "Vector<storage.fileJpeg>"},
	}
	for _, tt := range tests {
		got := stripNamespace(tt.input)
		if got != tt.want {
			t.Errorf("stripNamespace(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestNormalizeVectorType(t *testing.T) {
	tests := []struct {
		input, want string
	}{
		{"vector<int>", "Vector<int>"},
		{"Vector<string>", "Vector<string>"},
		{"int", "int"},
	}
	for _, tt := range tests {
		got := normalizeVectorType(tt.input)
		if got != tt.want {
			t.Errorf("normalizeVectorType(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestResolveGoTypeForField(t *testing.T) {
	tests := []struct {
		arg  Arg
		want string
	}{
		{Arg{Name: "val", Type: "string", FlagBit: -1}, "string"},
		{Arg{Name: "opt", Type: "string", FlagBit: 0}, "string"},
		{Arg{Name: "opt_int", Type: "int", FlagBit: 1}, "int32"},
		{Arg{Name: "opt_bool", Type: "Bool", FlagBit: 2}, "bool"},
		{Arg{Name: "obj", Type: "Photo", FlagBit: -1}, "*Photo"},
	}
	for _, tt := range tests {
		got := resolveGoTypeForField(tt.arg, "types", nil, nil)
		if got != tt.want {
			t.Errorf("resolveGoTypeForField(%+v) = %q, want %q", tt.arg, got, tt.want)
		}
	}
}

func TestResolveGoTypeForField_PointerOptional(t *testing.T) {
	arg := Arg{Name: "opt_bytes", Type: "bytes", FlagBit: 3}
	got := resolveGoTypeForField(arg, "types", nil, nil)
	if got != "[]byte" {
		t.Errorf("resolveGoTypeForField optional bytes = %q, want %q", got, "[]byte")
	}
}

func TestBuildTypeData(t *testing.T) {
	c := Combinator{
		QualName: "testObj",
		Name:     "TestObj",
		ID:       0xDEADBEEF,
		Args: []Arg{
			{Name: "flags", Type: "#", FlagBit: -1},
			{Name: "val", Type: "int", FlagBit: -1},
			{Name: "opt_str", Type: "string", FlagBit: 0, FlagName: "flags"},
		},
		Type: "TestObj",
	}
	td := buildTypeData(c, "types", nil, nil, nil)
	if td.Name != "TestObj" {
		t.Errorf("td.Name = %q, want %q", td.Name, "TestObj")
	}
	if td.ID != 0xDEADBEEF {
		t.Errorf("td.ID = 0x%08x, want 0xDEADBEEF", td.ID)
	}
	if !td.HasFlags {
		t.Error("td.HasFlags = false, want true")
	}
	if len(td.Fields) != 3 {
		t.Fatalf("len(td.Fields) = %d, want 3", len(td.Fields))
	}
	if td.Fields[0].Name != "Flags" {
		t.Errorf("Fields[0].Name = %q, want Flags", td.Fields[0].Name)
	}
	if !td.Fields[0].IsFlags {
		t.Error("Fields[0].IsFlags = false, want true")
	}
	if td.Fields[1].GoType != "int32" {
		t.Errorf("Fields[1].GoType = %q, want int32", td.Fields[1].GoType)
	}
	if len(td.FlagSyncs) != 1 {
		t.Fatalf("len(td.FlagSyncs) = %d, want 1", len(td.FlagSyncs))
	}
	if td.FlagSyncs[0].Field != "OptStr" {
		t.Errorf("FlagSyncs[0].Field = %q, want OptStr", td.FlagSyncs[0].Field)
	}
}

func TestComputeKnownTypes(t *testing.T) {
	combos := []Combinator{
		{Section: SectionTypes, Type: "User", QualName: "user"},
		{Section: SectionTypes, Type: "User", QualName: "userEmpty"},
		{Section: SectionFunctions, Type: "User", QualName: "getUser"},
	}
	m := computeKnownTypes(combos)
	if !m["User"] {
		t.Error("expected User in known types")
	}
	if !m["user"] {
		t.Error("expected user in known types")
	}
	if !m["userEmpty"] {
		t.Error("expected userEmpty in known types")
	}
	if m["getUser"] {
		t.Error("getUser should not be in known types (function)")
	}
}

func TestResolveReturnType(t *testing.T) {
	baseTypes := map[string]bool{"User": true}
	typeToConstructor := map[string][]Combinator{
		"Chat": {{QualName: "chat", Section: SectionTypes}},
	}

	tests := []struct {
		c       Combinator
		goType  string
		isBool  bool
		isVec   bool
	}{
		{Combinator{Type: "Bool"}, "bool", true, false},
		{Combinator{Type: "true"}, "bool", true, false},
		{Combinator{Type: "int"}, "int32", false, false},
		{Combinator{Type: "long"}, "int64", false, false},
		{Combinator{Type: "double"}, "float64", false, false},
		{Combinator{Type: "string"}, "string", false, false},
		{Combinator{Type: "bytes"}, "[]byte", false, false},
		{Combinator{Type: "User"}, "UserClass", false, false},
		{Combinator{Type: "Chat"}, "*Chat", false, false},
		{Combinator{Type: "Vector<int>"}, "TLObject", false, true},
		{Combinator{Type: "Unknown"}, "TLObject", false, false},
	}

	for _, tt := range tests {
		rt := resolveReturnType(tt.c, baseTypes, typeToConstructor, nil)
		if rt.GoType != tt.goType {
			t.Errorf("resolveReturnType(%q).GoType = %q, want %q", tt.c.Type, rt.GoType, tt.goType)
		}
		if rt.IsBool != tt.isBool {
			t.Errorf("resolveReturnType(%q).IsBool = %v, want %v", tt.c.Type, rt.IsBool, tt.isBool)
		}
		if rt.IsVector != tt.isVec {
			t.Errorf("resolveReturnType(%q).IsVector = %v, want %v", tt.c.Type, rt.IsVector, tt.isVec)
		}
	}
}

func TestResolveReturnType_MultiConstructor(t *testing.T) {
	baseTypes := map[string]bool{"User": true}
	typeToConstructor := map[string][]Combinator{
		"User": {{QualName: "user"}, {QualName: "userEmpty"}},
	}
	c := Combinator{Type: "User"}
	rt := resolveReturnType(c, baseTypes, typeToConstructor, nil)
	if rt.GoType != "UserClass" {
		t.Errorf("resolveReturnType multi = %q, want UserClass", rt.GoType)
	}
}

func TestResolveReturnType_UsesQualType(t *testing.T) {
	c := Combinator{Type: "", QualType: "Bool"}
	rt := resolveReturnType(c, nil, nil, nil)
	if !rt.IsBool {
		t.Error("expected IsBool when Type is empty and QualType is Bool")
	}
}

func TestCleanupGeneratedFiles_NonexistentDir(t *testing.T) {
	err := cleanupGeneratedFiles("/tmp/nonexistent_dir_for_test_12345", []string{"*.go"})
	if err != nil {
		t.Errorf("cleanupGeneratedFiles on nonexistent dir returned error: %v", err)
	}
}

func TestFlagSyncCondition(t *testing.T) {
	tests := []struct {
		fs   flagSync
		want string
	}{
		{flagSync{Field: "Val", GoType: "bool"}, "v.Val"},
		{flagSync{Field: "Name", GoType: "string"}, "v.Name != \"\""},
		{flagSync{Field: "Count", GoType: "int32"}, "v.Count != 0"},
		{flagSync{Field: "ID", GoType: "int64"}, "v.ID != 0"},
		{flagSync{Field: "Score", GoType: "float64"}, "v.Score != 0"},
		{flagSync{Field: "Flags", GoType: "uint32"}, "v.Flags != 0"},
		{flagSync{Field: "Obj", GoType: "*User"}, "v.Obj != nil"},
		{flagSync{Field: "Items", GoType: "[]int32"}, "v.Items != nil"},
		{flagSync{Field: "C", GoType: "UserClass"}, "v.C != nil"},
	}
	for _, tt := range tests {
		got := flagSyncCondition(tt.fs)
		if got != tt.want {
			t.Errorf("flagSyncCondition(%+v) = %q, want %q", tt.fs, got, tt.want)
		}
	}
}

func readFile(dir, name string) (string, error) {
	data, err := readFileGlob(dir, name)
	if err != nil {
		return "", err
	}
	return data, nil
}

func readFileGlob(dir, pattern string) (string, error) {
	matches, err := filepath.Glob(filepath.Join(dir, pattern))
	if err != nil {
		return "", err
	}
	if len(matches) == 0 {
		return "", fmt.Errorf("no files matching %q in %q", pattern, dir)
	}
	var content string
	for _, f := range matches {
		data, err := os.ReadFile(f)
		if err != nil {
			return "", err
		}
		content += string(data)
	}
	return content, nil
}
