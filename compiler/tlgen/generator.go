package tlgen

import (
	"fmt"
	"go/format"
	"maps"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// ---------------------------------------------------------------------------
// Utility functions
// ---------------------------------------------------------------------------

func indentCode(code, indent string) string {
	lines := strings.Split(code, "\n")
	for i, line := range lines {
		if strings.TrimSpace(line) != "" {
			lines[i] = indent + line
		}
	}
	return strings.Join(lines, "\n")
}

func classifyTypeDomain(typeName string, constructors []Combinator) string {
	nsCounts := map[string]int{}
	for _, c := range constructors {
		if c.Namespace != "" {
			nsCounts[c.Namespace]++
		}
	}
	if len(nsCounts) == 1 {
		for ns := range nsCounts {
			return ns
		}
	}
	if len(nsCounts) > 1 {
		names := make([]string, 0, len(nsCounts))
		for ns := range nsCounts {
			names = append(names, ns)
		}
		sort.Strings(names)
		best := ""
		bestN := 0
		for _, ns := range names {
			if n := nsCounts[ns]; n > bestN {
				best = ns
				bestN = n
			}
		}
		if best != "" {
			return best
		}
	}

	lower := strings.ToLower(typeName)

	keywords := []struct {
		keys   []string
		domain string
	}{
		{[]string{"auth", "login", "sentcode", "password", "passkey", "authorization"}, "auth"},
		{[]string{"message", "dialog", "sticker", "emoji", "reaction", "forum", "quickreply", "todo", "sponsored"}, "messages"},
		{[]string{"peer", "user", "chat", "channel", "contact", "participant", "invite", "username", "admin"}, "peer"},
		{[]string{"document", "photo", "video", "audio", "upload", "webpage", "maskcoord", "videosize", "filesize"}, "media"},
		{[]string{"update", "difference", "pts"}, "updates"},
		{[]string{"account", "privacy", "notify", "contentsetting", "autodownload", "autosave", "wallpaper", "business", "birthday", "connectedbot"}, "account"},
		{[]string{"bot", "keyboard", "inline", "game", "attachmenu", "webview", "botmenu"}, "bots"},
		{[]string{"payment", "star", "invoice", "shipping", "premium", "boost", "giveaway", "bankcard", "stargift"}, "payments"},
		{[]string{"phone", "call", "groupcall", "conference"}, "phone"},
		{[]string{"story", "stories"}, "stories"},
		{[]string{"page", "richtext", "textblock"}, "pages"},
		{[]string{"poll", "vote"}, "polls"},
		{[]string{"chatlist", "folder", "dialogfilter"}, "chatlists"},
		{[]string{"encrypt", "decrypt", "dhinner", "pqinner"}, "crypto"},
		{[]string{"geo", "location"}, "geo"},
		{[]string{"config", "dcoption", "apponfig", "langpack", "country", "timezone", "invite", "nearestdc", "deeplink", "promodata", "termsofservice", "support", "apdate", "cdnconfig", "httwait", "req_pq", "res_pq", "ping", "pong", "future_salt", "msg_detailed", "msg_resend", "msgs_", "rpc_", "destroy_", "new_session", "bad_", "bind_auth", "init_connection", "invoke_", "set_client", "client_dh", "server_dh"}, "core"},
		{[]string{"sms_job", "smsjob"}, "smsjobs"},
		{[]string{"ai_compose", "aicompose"}, "aicompose"},
	}

	for _, kw := range keywords {
		for _, k := range kw.keys {
			if strings.Contains(lower, k) {
				return kw.domain
			}
		}
	}
	return "core"
}

func classifyFuncDomain(c Combinator) string {
	if c.Namespace != "" {
		return c.Namespace
	}
	return "core"
}

func domainFileOrder(domains map[string]bool) []string {
	order := []string{
		"auth", "messages", "peer", "media", "updates", "account",
		"bots", "payments", "phone", "stories", "chatlists",
		"pages", "polls", "geo", "crypto", "smsjobs", "aicompose",
		"system", "core",
	}
	var result []string
	for _, d := range order {
		if domains[d] {
			result = append(result, d)
		}
	}

	var rest []string
	for d := range domains {
		found := false
		for _, o := range order {
			if o == d {
				found = true
				break
			}
		}
		if !found {
			rest = append(rest, d)
		}
	}
	sort.Strings(rest)
	result = append(result, rest...)
	return result
}

func writeGoFile(path string, content string) error {
	formatted, err := format.Source([]byte(content))
	if err != nil {
		if os.Getenv("TLGEN_DEBUG") != "" {
			_ = os.WriteFile(path+".raw", []byte(content), 0o644)
		}
		return fmt.Errorf("format error in %s: %w", path, err)
	}
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, formatted, 0o644)
}

func writeImports(buf *strings.Builder, pkg ...string) {
	buf.WriteString("import (\n")
	for _, p := range pkg {
		buf.WriteString("\t\"" + p + "\"\n")
	}
	buf.WriteString(")\n\n")
}

func cleanupGeneratedFiles(outDir string, patterns []string) error {
	entries, err := os.ReadDir(outDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		for _, p := range patterns {
			if match, _ := filepath.Match(p, name); match {
				os.Remove(filepath.Join(outDir, name))
				break
			}
		}
	}
	return nil
}

func flagSyncCondition(fs flagSync) string {
	switch fs.GoType {
	case "bool":
		return fmt.Sprintf("v.%s", fs.Field)
	case "string":
		return fmt.Sprintf("v.%s != \"\"", fs.Field)
	case "int32", "int64", "float64", "uint32":
		return fmt.Sprintf("v.%s != 0", fs.Field)
	default:
		// Assumes a nil-able field type (pointer, slice, map, interface).
		// A hypothetical non-pointer fixed-array optional would emit invalid Go
		// here; no current TL schema input produces one.
		return fmt.Sprintf("v.%s != nil", fs.Field)
	}
}

func domainTitle(domain string) string {
	if domain == "" {
		return "Core"
	}
	return strings.ToUpper(domain[:1]) + domain[1:]
}

// ---------------------------------------------------------------------------
// genConfig — captures all systematic differences between the tg and e2e
// package generation targets.
// ---------------------------------------------------------------------------

type genConfig struct {
	pkgName     string   // "tg" or "e2e"
	baseImports []string // extra imports for types files (beyond "bytes")
	funcImports []string // extra imports for function files (beyond "bytes", "context")
	reader      string   // "*Reader" or "*tg.Reader"
	tlObject    string   // "TLObject" or "tg.TLObject"
	writeInt    string   // "WriteInt" or "tg.WriteInt"
	readTLFunc  string   // "ReadTLObject" or "ReadE2ETLObject"
	invokerType string   // "Invoker" or "tg.Invoker"

	xformWrite func(string) string // transform encode code snippets
	xformRead  func(string) string // transform decode code snippets
	xformType  func(string) string // transform field Go types

	// behavioral flags
	perDomainConstructors bool     // generate per-domain ConstructorMap{Domain} files
	namesMapTypesOnly     bool     // filter NamesMap to SectionTypes only
	verboseInvokeDocs     bool     // long doc comment on invoke methods
	structDocLink         bool     // include "See https://..." link in type struct doc
	clientScope           string   // doc-comment scope for RPCClient: "" (tg) or "e2e " (e2e)
	namesMapDocExample    bool     // include "(e.g. messages.sendMessage)" in NamesMap doc
	funcCleanupPatterns   []string // cleanup patterns for function generation
}

func tgConfig() genConfig {
	return genConfig{
		pkgName:               "tg",
		reader:                "*Reader",
		tlObject:              "TLObject",
		writeInt:              "WriteInt",
		readTLFunc:            "ReadTLObject",
		invokerType:           "Invoker",
		xformWrite:            func(s string) string { return s },
		xformRead:             func(s string) string { return s },
		xformType:             func(s string) string { return s },
		perDomainConstructors: true,
		verboseInvokeDocs:     true,
		structDocLink:         true,
		clientScope:           "",
		namesMapDocExample:    true,
		funcCleanupPatterns:   []string{"tl_*_gen.go", "*_methods_gen.go"},
	}
}

func e2eConfig() genConfig {
	return genConfig{
		pkgName:             "e2e",
		baseImports:         []string{"github.com/mtgo-labs/mtgo/tg"},
		funcImports:         []string{"github.com/mtgo-labs/mtgo/tg"},
		reader:              "*tg.Reader",
		tlObject:            "tg.TLObject",
		writeInt:            "tg.WriteInt",
		readTLFunc:          "ReadE2ETLObject",
		invokerType:         "tg.Invoker",
		xformWrite:          e2ePrefixWriteCode,
		xformRead:           e2ePrefixReadCode,
		xformType:           prefixE2EType,
		namesMapTypesOnly:   true,
		structDocLink:       false,
		clientScope:         "e2e ",
		namesMapDocExample:  false,
		funcCleanupPatterns: []string{"*_methods_gen.go"},
	}
}

// ---------------------------------------------------------------------------
// Public API — thin wrappers that select a config
// ---------------------------------------------------------------------------

func GeneratePackageFiles(outDir, pkgName string, layer int) error {
	docContent := "// Code generated by tlgen. DO NOT EDIT.\n\n// Package " + pkgName + " provides auto-generated Go types for the Telegram MTProto TL schema.\n//\n// Each type implements the tg.TLObject interface with Encode and ConstructorID methods.\n// Use Decode{Name} functions to deserialize types from a reader.\n// Use tg.Registry to look up decoders by constructor ID.\npackage " + pkgName + "\n"
	return writeGoFile(filepath.Join(outDir, "doc.go"), docContent)
}

func GenerateGroupedTypes(outDir string, combos []Combinator, layer int) error {
	return generateGroupedTypes(tgConfig(), outDir, combos, layer)
}

func GenerateGroupedFunctions(outDir string, combos []Combinator, layer int) error {
	return generateGroupedFunctions(tgConfig(), outDir, combos, layer)
}

func GenerateGroupedConstructors(outDir string, combos []Combinator) error {
	return generateGroupedConstructors(tgConfig(), outDir, combos)
}

func GenerateNamesMap(outDir string, combos []Combinator) error {
	return generateNamesMap(tgConfig(), outDir, combos)
}

func GenerateFunctionsMap(outDir string, combos []Combinator) error {
	return generateFunctionsMap(tgConfig(), outDir, combos)
}

func GenerateE2EPackage(outDir string, combos []Combinator, layer int) error {
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return err
	}

	docContent := "// Code generated by tlgen. DO NOT EDIT.\n\n// Package e2e provides auto-generated Go types for the Telegram end-to-end encrypted (secret chat) TL schema.\n//\n// Each type implements the tg.TLObject interface with Encode and ConstructorID methods.\n// Use Decode{Name} functions to deserialize types from a reader.\npackage e2e\n"
	if err := writeGoFile(filepath.Join(outDir, "doc.go"), docContent); err != nil {
		return err
	}

	cfg := e2eConfig()

	if err := generateE2ERegistry(outDir); err != nil {
		return err
	}
	if err := generateGroupedTypes(cfg, outDir, combos, layer); err != nil {
		return err
	}
	if err := generateGroupedConstructors(cfg, outDir, combos); err != nil {
		return err
	}
	if err := generateGroupedFunctions(cfg, outDir, combos, layer); err != nil {
		return err
	}
	if err := generateFunctionsMap(cfg, outDir, combos); err != nil {
		return err
	}
	return generateNamesMap(cfg, outDir, combos)
}

// ---------------------------------------------------------------------------
// Shared implementations
// ---------------------------------------------------------------------------

func generateGroupedTypes(cfg genConfig, outDir string, combos []Combinator, layer int) error {
	typeCombos := filterCombos(combos, SectionTypes)
	baseTypes := computeBaseTypes(combos)
	typeToConstructor := computeTypeToConstructor(combos)

	if err := cleanupGeneratedFiles(outDir, []string{"tl_*_gen.go", "*_types_gen.go"}); err != nil {
		return err
	}

	typeMap := map[string][]Combinator{}
	typeOrder := []string{}
	for _, c := range typeCombos {
		if _, exists := typeMap[c.Type]; !exists {
			typeOrder = append(typeOrder, c.Type)
		}
		typeMap[c.Type] = append(typeMap[c.Type], c)
	}

	typeToDomain := map[string]string{}
	for _, typeName := range typeOrder {
		typeToDomain[typeName] = classifyTypeDomain(typeName, typeMap[typeName])
	}

	domainTypes := map[string][]string{}
	domainSet := map[string]bool{}
	for _, typeName := range typeOrder {
		domain := typeToDomain[typeName]
		domainTypes[domain] = append(domainTypes[domain], typeName)
		domainSet[domain] = true
	}

	allNames := map[string]bool{}

	for _, domain := range domainFileOrder(domainSet) {
		typeNames := domainTypes[domain]

		var buf strings.Builder
		buf.WriteString("// Code generated by tlgen. DO NOT EDIT.\n\npackage " + cfg.pkgName + "\n\n")
		writeImports(&buf, append([]string{"bytes"}, cfg.baseImports...)...)

		for _, typeName := range typeNames {
			constructors := typeMap[typeName]

			domainAllNames := map[string]bool{}
			maps.Copy(domainAllNames, allNames)

			var types []typeTemplateData
			for _, c := range constructors {
				td := buildTypeData(c, "types", baseTypes, typeToConstructor, domainAllNames)
				types = append(types, td)
			}

			basePascal := SnakeToPascal(typeName)
			isMulti := len(types) > 1

			if isMulti {
				fmt.Fprintf(&buf, "\n// %s is the interface for TL type %s.\n// Implementations must satisfy %s and are used to represent\n// any constructor of the %s TL type.\ntype %s interface {\n\t%s\n\tis%s()\n}\n", basePascal+"Class", basePascal, cfg.tlObject, basePascal, basePascal+"Class", cfg.tlObject, basePascal)
			}

			for _, td := range types {
				constName := td.Name + "TypeID"
				fmt.Fprintf(&buf, "\n// %s is the constructor ID for TL type %s.\nconst %s = 0x%08x\n", constName, td.QualName, constName, td.ID)
			}

			for _, td := range types {
				if isMulti {
					fmt.Fprintf(&buf, "\n// is%s marks %s as implementing the %sClass interface.\nfunc (*%s) is%s() {}\n", basePascal, td.Name, basePascal, td.Name, basePascal)
				}
			}

			for _, td := range types {
				writeTypeStruct(&buf, cfg, td)
				writeSetFlags(&buf, td.Name, td, td.HasFlags)
				constName := td.Name + "TypeID"
				fmt.Fprintf(&buf, "\n// ConstructorID returns the TL constructor identifier 0x%08x.\nfunc (v *%s) ConstructorID() uint32 {\n\treturn %s\n}\n", td.ID, td.Name, constName)
				writeEncodeMethod(&buf, cfg, td.Name, constName, td)
				writeDecodeMethod(&buf, cfg, td.Name, td)
				fmt.Fprintf(&buf, "\nfunc init() {\n\tRegistry[%s] = func(r %s) (%s, error) {\n\t\treturn Decode%s(r)\n\t}\n}\n", constName, cfg.reader, cfg.tlObject, td.Name)
			}

			maps.Copy(allNames, domainAllNames)
		}

		fileName := fmt.Sprintf("%s_types_gen.go", domain)
		if err := writeGoFile(filepath.Join(outDir, fileName), buf.String()); err != nil {
			return err
		}
	}

	return nil
}

func generateGroupedFunctions(cfg genConfig, outDir string, combos []Combinator, layer int) error {
	funcCombos := filterCombos(combos, SectionFunctions)
	if len(funcCombos) == 0 {
		return nil
	}
	baseTypes := computeBaseTypes(combos)
	typeToConstructor := computeTypeToConstructor(combos)
	knownTypes := computeKnownTypes(combos)

	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return err
	}

	if err := cleanupGeneratedFiles(outDir, cfg.funcCleanupPatterns); err != nil {
		return err
	}

	namespaceMap := map[string][]Combinator{}
	namespaceOrder := []string{}
	for _, c := range funcCombos {
		ns := classifyFuncDomain(c)
		if _, exists := namespaceMap[ns]; !exists {
			namespaceOrder = append(namespaceOrder, ns)
		}
		namespaceMap[ns] = append(namespaceMap[ns], c)
	}

	allNames := map[string]bool{}

	for _, ns := range namespaceOrder {
		nsCombos := namespaceMap[ns]

		// needsFmt mirrors generateInvokeMethod's fmt.Errorf emit condition
		// (!IsBool && !IsVector && GoType != "TLObject") so the import is added
		// exactly when fmt is referenced.
		needsFmt := false
		for _, c := range nsCombos {
			rt := resolveReturnType(c, baseTypes, typeToConstructor, knownTypes)
			if !rt.IsBool && !rt.IsVector && rt.GoType != "TLObject" {
				needsFmt = true
				break
			}
		}

		var buf strings.Builder
		buf.WriteString("// Code generated by tlgen. DO NOT EDIT.\n\npackage " + cfg.pkgName + "\n\n")

		imports := []string{"bytes", "context"}
		imports = append(imports, cfg.funcImports...)
		if needsFmt {
			imports = append(imports, "fmt")
		}
		buf.WriteString("import (\n")
		for _, imp := range imports {
			fmt.Fprintf(&buf, "\t%q\n", imp)
		}
		buf.WriteString(")\n")

		nsAllNames := map[string]bool{}
		maps.Copy(nsAllNames, allNames)

		for _, c := range nsCombos {
			groupAllNames := map[string]bool{}
			maps.Copy(groupAllNames, nsAllNames)

			td := buildTypeData(c, "functions", baseTypes, typeToConstructor, groupAllNames)
			constName := td.Name + "TypeID"
			retType := resolveReturnType(c, baseTypes, typeToConstructor, knownTypes)
			structName := td.Name + "Request"

			tlMethod := td.QualName
			methodPath := strings.ReplaceAll(tlMethod, ".", "/")
			fmt.Fprintf(&buf, "\n// %s is the constructor ID for the RPC function %s.\nconst %s = 0x%08x\n", constName, tlMethod, constName, td.ID)

			fmt.Fprintf(&buf, "\n// %s represents TL type `%s#%08x`.\n//\n// See https://core.telegram.org/method/%s for reference.\ntype %s struct {\n", structName, tlMethod, td.ID, methodPath, structName)
			writeTypeFields(&buf, cfg, td.Fields)
			buf.WriteString("}\n")

			writeSetFlags(&buf, structName, td, len(td.FlagSyncs) > 0)
			fmt.Fprintf(&buf, "\n// ConstructorID returns the TL constructor identifier 0x%08x.\nfunc (v *%s) ConstructorID() uint32 {\n\treturn %s\n}\n", td.ID, structName, constName)

			writeEncodeMethod(&buf, cfg, structName, constName, td)
			generateInvokeMethod(&buf, cfg, td, retType, structName)

			maps.Copy(nsAllNames, groupAllNames)
		}

		fileName := fmt.Sprintf("%s_methods_gen.go", ns)
		if err := writeGoFile(filepath.Join(outDir, fileName), buf.String()); err != nil {
			return err
		}

		maps.Copy(allNames, nsAllNames)
	}

	clientBuf := generateClientBoilerplate(cfg)
	return writeGoFile(filepath.Join(outDir, "tl_client_gen.go"), clientBuf.String())
}

func generateGroupedConstructors(cfg genConfig, outDir string, combos []Combinator) error {
	if err := cleanupGeneratedFiles(outDir, []string{"*_constructors_gen.go"}); err != nil {
		return err
	}

	typeCombos := filterCombos(combos, SectionTypes)
	baseTypes := computeBaseTypes(combos)
	typeToConstructor := computeTypeToConstructor(combos)

	typeMap := map[string][]Combinator{}
	typeOrder := []string{}
	for _, c := range typeCombos {
		if _, exists := typeMap[c.Type]; !exists {
			typeOrder = append(typeOrder, c.Type)
		}
		typeMap[c.Type] = append(typeMap[c.Type], c)
	}

	typeToDomain := map[string]string{}
	for _, typeName := range typeOrder {
		typeToDomain[typeName] = classifyTypeDomain(typeName, typeMap[typeName])
	}

	domainTypes := map[string][]string{}
	domainSet := map[string]bool{}
	for _, typeName := range typeOrder {
		domain := typeToDomain[typeName]
		domainTypes[domain] = append(domainTypes[domain], typeName)
		domainSet[domain] = true
	}

	allNames := map[string]bool{}

	// Per-domain files (tg only).
	if cfg.perDomainConstructors {
		for _, domain := range domainFileOrder(domainSet) {
			typeNames := domainTypes[domain]

			var buf strings.Builder
			buf.WriteString("// Code generated by tlgen. DO NOT EDIT.\n\npackage " + cfg.pkgName + "\n\n")
			fmt.Fprintf(&buf, "// ConstructorMap%s maps constructor IDs to factory functions.\nvar ConstructorMap%s = map[uint32]func() %s{\n", domainTitle(domain), domainTitle(domain), cfg.tlObject)

			for _, typeName := range typeNames {
				constructors := typeMap[typeName]
				groupAllNames := map[string]bool{}
				maps.Copy(groupAllNames, allNames)

				for _, c := range constructors {
					td := buildTypeData(c, "types", baseTypes, typeToConstructor, groupAllNames)
					fmt.Fprintf(&buf, "\t0x%08x: func() %s { return &%s{} },\n", td.ID, cfg.tlObject, td.Name)
				}

				maps.Copy(allNames, groupAllNames)
			}

			buf.WriteString("}\n")

			fileName := fmt.Sprintf("%s_constructors_gen.go", domain)
			if err := writeGoFile(filepath.Join(outDir, fileName), buf.String()); err != nil {
				return err
			}
		}
	}

	// Union file.
	var unionBuf strings.Builder
	unionBuf.WriteString("// Code generated by tlgen. DO NOT EDIT.\n\npackage " + cfg.pkgName + "\n\n")
	if len(cfg.baseImports) > 0 {
		writeImports(&unionBuf, cfg.baseImports...)
	}
	fmt.Fprintf(&unionBuf, "// ConstructorMap maps constructor IDs to factory functions that return zero-value TLObjects.\nvar ConstructorMap = map[uint32]func() %s{\n", cfg.tlObject)

	for _, domain := range domainFileOrder(domainSet) {
		fmt.Fprintf(&unionBuf, "\t// %s types\n", domain)
		typeNames := domainTypes[domain]
		groupAllNames := map[string]bool{}
		maps.Copy(groupAllNames, allNames)

		for _, typeName := range typeNames {
			constructors := typeMap[typeName]
			for _, c := range constructors {
				td := buildTypeData(c, "types", baseTypes, typeToConstructor, groupAllNames)
				fmt.Fprintf(&unionBuf, "\t0x%08x: func() %s { return &%s{} },\n", td.ID, cfg.tlObject, td.Name)
			}
		}

		maps.Copy(allNames, groupAllNames)
	}

	unionBuf.WriteString("}\n")
	return writeGoFile(filepath.Join(outDir, "tl_constructors_gen.go"), unionBuf.String())
}

func generateNamesMap(cfg genConfig, outDir string, combos []Combinator) error {
	var buf strings.Builder
	buf.WriteString("// Code generated by tlgen. DO NOT EDIT.\n\npackage " + cfg.pkgName + "\n\n")
	if cfg.namesMapDocExample {
		buf.WriteString("// NamesMap maps TL qualified names (e.g. \"messages.sendMessage\") to their constructor IDs.\n")
	} else {
		buf.WriteString("// NamesMap maps TL qualified names to their constructor IDs.\n")
	}
	buf.WriteString("var NamesMap = map[string]uint32{\n")

	for _, c := range combos {
		if cfg.namesMapTypesOnly && c.Section != SectionTypes {
			continue
		}
		fmt.Fprintf(&buf, "\t%q: 0x%08x,\n", c.QualName, c.ID)
	}

	buf.WriteString("}\n")
	return writeGoFile(filepath.Join(outDir, "tl_names_gen.go"), buf.String())
}

func generateFunctionsMap(cfg genConfig, outDir string, combos []Combinator) error {
	funcCombos := filterCombos(combos, SectionFunctions)
	if len(funcCombos) == 0 {
		return nil
	}
	baseTypes := computeBaseTypes(combos)
	typeToConstructor := computeTypeToConstructor(combos)
	allNames := map[string]bool{}

	var buf strings.Builder
	buf.WriteString("// Code generated by tlgen. DO NOT EDIT.\n\npackage " + cfg.pkgName + "\n\n")
	if len(cfg.baseImports) > 0 {
		writeImports(&buf, cfg.baseImports...)
	}
	buf.WriteString("// FunctionsMap maps function constructor IDs to factory functions that return zero-value request objects.\n")
	fmt.Fprintf(&buf, "var FunctionsMap = map[uint32]func() %s{\n", cfg.tlObject)

	for _, c := range funcCombos {
		td := buildTypeData(c, "functions", baseTypes, typeToConstructor, allNames)
		structName := td.Name + "Request"
		fmt.Fprintf(&buf, "\t0x%08x: func() %s { return &%s{} },\n", td.ID, cfg.tlObject, structName)
	}

	buf.WriteString("}\n")
	return writeGoFile(filepath.Join(outDir, "tl_functions_gen.go"), buf.String())
}

// ---------------------------------------------------------------------------
// Shared code-emit helpers
// ---------------------------------------------------------------------------

func writeTypeStruct(buf *strings.Builder, cfg genConfig, td typeTemplateData) {
	if cfg.structDocLink {
		fmt.Fprintf(buf, "\n// %s represents the TL constructor %s (0x%08x).\n//\n// See https://core.telegram.org/constructor/%s for reference.\ntype %s struct {\n", td.Name, td.QualName, td.ID, td.QualName, td.Name)
	} else {
		fmt.Fprintf(buf, "\n// %s represents the TL constructor %s (0x%08x).\ntype %s struct {\n", td.Name, td.QualName, td.ID, td.Name)
	}
	writeTypeFields(buf, cfg, td.Fields)
	buf.WriteString("}\n")
}

func writeTypeFields(buf *strings.Builder, cfg genConfig, fields []fieldData) {
	for _, f := range fields {
		goType := cfg.xformType(f.GoType)
		if f.IsFlags {
			fmt.Fprintf(buf, "\t%s %s `json:\"-\"`\n", f.Name, goType)
		} else {
			fmt.Fprintf(buf, "\t%s %s `json:\"%s,omitempty\"`\n", f.Name, goType, f.JSONTag)
		}
	}
}

func writeSetFlags(buf *strings.Builder, name string, td typeTemplateData, shouldEmit bool) {
	if !shouldEmit {
		return
	}
	fmt.Fprintf(buf, "\n// SetFlags computes flags from non-zero optional fields.\nfunc (v *%s) SetFlags() {\n", name)
	for _, fs := range td.FlagSyncs {
		fmt.Fprintf(buf, "\tif %s {\n\t\tv.%s.Set(%d)\n\t}\n", flagSyncCondition(fs), fs.FlagName, fs.Bit)
	}
	buf.WriteString("}\n")
}

func writeEncodeMethod(buf *strings.Builder, cfg genConfig, name, constName string, td typeTemplateData) {
	fmt.Fprintf(buf, "\n// Encode serializes %s to a bytes.Buffer using the TL binary protocol.\nfunc (v *%s) Encode(b *bytes.Buffer) error {\n\t%s(b, %s)\n", name, name, cfg.writeInt, constName)
	if td.HasFlags {
		buf.WriteString("\tv.SetFlags()\n")
		for _, f := range td.Fields {
			if f.IsFlags {
				fmt.Fprintf(buf, "\t%s(b, uint32(v.%s))\n", cfg.writeInt, f.Name)
			}
		}
	}
	for _, wl := range td.WriteLines {
		if wl.IsFlags || wl.Code == "" {
			continue
		}
		code := cfg.xformWrite(wl.Code)
		if wl.IsGuarded {
			fmt.Fprintf(buf, "\tif v.%s.Has(%d) {\n\t\t%s\n\t}\n", wl.FlagName, wl.FlagBit, code)
		} else {
			fmt.Fprintf(buf, "\t%s\n", code)
		}
	}
	buf.WriteString("\treturn nil\n}\n")
}

func writeDecodeMethod(buf *strings.Builder, cfg genConfig, name string, td typeTemplateData) {
	fmt.Fprintf(buf, "\n// Decode%s deserializes a %s from a reader using the TL binary protocol.\nfunc Decode%s(r %s) (*%s, error) {\n\tv := &%s{}\n", name, name, name, cfg.reader, name, name)
	for _, rl := range td.ReadLines {
		code := cfg.xformRead(rl.Code)
		buf.WriteString(indentCode(code, "\t") + "\n")
	}
	buf.WriteString("\treturn v, nil\n}\n")
}

func generateInvokeMethod(buf *strings.Builder, cfg genConfig, td typeTemplateData, retType returnType, structName string) {
	hasArgs := len(td.Fields) > 0
	methodName := td.Name
	qualName := td.QualName
	requestName := structName

	isGenericReturn := retType.GoType == "TLObject"
	returnTypeStr := retType.GoType
	if isGenericReturn {
		returnTypeStr = cfg.tlObject
	}

	if cfg.verboseInvokeDocs {
		fmt.Fprintf(buf, "\n// %s invokes the %s RPC method on the server.\n//\n// Parameters:\n//   - ctx: context for cancellation and timeout\n//   - req: the request parameters\n//\n// Returns the result of the RPC call, or an error if the invocation fails.\nfunc (c *RPCClient) %s(ctx context.Context", methodName, qualName, methodName)
	} else {
		fmt.Fprintf(buf, "\n// %s invokes the %s RPC method on the server.\nfunc (c *RPCClient) %s(ctx context.Context", methodName, qualName, methodName)
	}
	if hasArgs {
		fmt.Fprintf(buf, ", req *%s", requestName)
	}
	fmt.Fprintf(buf, ") (%s, error) {\n", returnTypeStr)

	invokeBody := func(reqExpr string) {
		fmt.Fprintf(buf, "\tresult, err := c.invoke(ctx, %s, func(r %s) (%s, error) {\n\t\treturn %s(r)\n\t})\n", reqExpr, cfg.reader, cfg.tlObject, cfg.readTLFunc)
	}

	reqExpr := "req"
	if !hasArgs {
		reqExpr = fmt.Sprintf("&%s{}", requestName)
	}

	if retType.IsBool {
		invokeBody(reqExpr)
		buf.WriteString("\tif err != nil {\n\t\treturn false, err\n\t}\n")
		buf.WriteString("\t_ = result\n\treturn true, nil\n")
	} else if retType.IsVector {
		invokeBody(reqExpr)
		buf.WriteString("\tif err != nil {\n\t\treturn nil, err\n\t}\n")
		buf.WriteString("\treturn result, nil\n")
	} else if isGenericReturn {
		invokeBody(reqExpr)
		buf.WriteString("\tif err != nil {\n\t\treturn nil, err\n\t}\n")
		buf.WriteString("\treturn result, nil\n")
	} else {
		invokeBody(reqExpr)
		buf.WriteString("\tif err != nil {\n\t\treturn nil, err\n\t}\n")
		fmt.Fprintf(buf, "\tif _c, _ok := result.(%s); _ok {\n\t\treturn _c, nil\n\t}\n\treturn nil, fmt.Errorf(\"unexpected result type %%T\", result)\n", retType.GoType)
	}
	buf.WriteString("}\n")
}

func generateClientBoilerplate(cfg genConfig) *strings.Builder {
	var buf strings.Builder
	buf.WriteString("// Code generated by tlgen. DO NOT EDIT.\n\npackage " + cfg.pkgName + "\n\n")
	writeImports(&buf, append([]string{"context"}, cfg.funcImports...)...)
	fmt.Fprintf(&buf, "// RPCClient provides typed RPC methods for all %sTL functions.\ntype RPCClient struct {\n\trpc %s\n}\n\n", cfg.clientScope, cfg.invokerType)
	fmt.Fprintf(&buf, "// NewRPCClient creates a new RPCClient.\nfunc NewRPCClient(rpc %s) *RPCClient {\n\treturn &RPCClient{rpc: rpc}\n}\n\n", cfg.invokerType)
	fmt.Fprintf(&buf, "func (c *RPCClient) invoke(ctx context.Context, req %s, decode func(%s) (%s, error)) (%s, error) {\n", cfg.tlObject, cfg.reader, cfg.tlObject, cfg.tlObject)
	buf.WriteString("\treturn c.rpc.RPCInvoke(ctx, req, decode)\n}")
	return &buf
}

// ---------------------------------------------------------------------------
// E2E-specific helpers
// ---------------------------------------------------------------------------

func generateE2ERegistry(outDir string) error {
	var buf strings.Builder
	buf.WriteString("// Code generated by tlgen. DO NOT EDIT.\n\npackage e2e\n\n")
	writeImports(&buf, "bytes", "github.com/mtgo-labs/mtgo/tg")
	buf.WriteString("// Registry maps TL constructor IDs to factory functions.\nvar Registry = map[uint32]func(*tg.Reader) (tg.TLObject, error){}\n\n")
	buf.WriteString("// ReadE2ETLObject reads a TLObject from r using the e2e Registry.\nfunc ReadE2ETLObject(r *tg.Reader) (tg.TLObject, error) {\n\tid, err := r.ReadUint32()\n\tif err != nil {\n\t\treturn nil, err\n\t}\n\tconstructor, ok := Registry[id]\n\tif !ok {\n\t\treturn nil, &tg.UnknownConstructorError{ID: id}\n\t}\n\treturn constructor(r)\n}\n\n")
	buf.WriteString("const vectorBareID uint32 = 0x1cb5c415\n\n")
	buf.WriteString("// EncodeTLObject encodes any TLObject to the buffer.\nfunc EncodeTLObject(b *bytes.Buffer, obj tg.TLObject) error {\n\treturn obj.Encode(b)\n}\n")
	return writeGoFile(filepath.Join(outDir, "e2e.go"), buf.String())
}

func e2ePrefixWriteCode(code string) string {
	replacements := []struct{ from, to string }{
		{"WriteInt(b,", "tg.WriteInt(b,"},
		{"WriteLong(b,", "tg.WriteLong(b,"},
		{"WriteString(b,", "tg.WriteString(b,"},
		{"WriteBytes(b,", "tg.WriteBytes(b,"},
		{"WriteDouble(b,", "tg.WriteDouble(b,"},
		{"WriteInt128(b,", "tg.WriteInt128(b,"},
		{"WriteInt256(b,", "tg.WriteInt256(b,"},
		{"WriteBool(b,", "tg.WriteBool(b,"},
		{"WriteVectorInt(b,", "tg.WriteVectorInt(b,"},
		{"WriteVectorLong(b,", "tg.WriteVectorLong(b,"},
		{"WriteVectorString(b,", "tg.WriteVectorString(b,"},
		{"WriteVectorBytes(b,", "tg.WriteVectorBytes(b,"},
		{"EncodeTLObject(b,", "tg.EncodeTLObject(b,"},
	}
	for _, r := range replacements {
		code = strings.ReplaceAll(code, r.from, r.to)
	}
	return code
}

func e2ePrefixReadCode(code string) string {
	replacements := []struct{ from, to string }{
		{"ReadTLObject(r)", "ReadE2ETLObject(r)"},
		{"checkVectorCount(", "tg.CheckVectorCount("},
		{"Fields(", "tg.Fields("},
	}
	for _, r := range replacements {
		code = strings.ReplaceAll(code, r.from, r.to)
	}
	return code
}
