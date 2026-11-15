// Command apicoverage generates the API coverage inventory for the high-level
// telegram package.
//
// It AST-scans the telegram package (domain method groups), telegram/types
// (bound method pairs), and telegram/params (option structs), then
// cross-references the TL schema (compiler/source/api.tl) to report which TL
// functions are wrapped by high-level operations.
//
// Output (generated files — never hand-edit):
//
//	go run ./cmd/apicoverage   # writes docs/api-coverage.md
//
// The inventory is derived entirely from source code and the schema; nothing
// is hand-maintained.
package main

import (
	"fmt"
	"go/ast"
	"go/doc"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const (
	repoRoot    = "."
	tlSchema    = "compiler/source/api.tl"
	tgDir       = "tg"
	telegramDir = "telegram"
	typesDir    = "telegram/types"
	paramsDir   = "telegram/params"
	outFile     = "docs/api-coverage.md"
)

// method is one high-level client operation.
type method struct {
	Name string
	Doc  string
	Sig  string
}

// boundType is a telegram/types type with bound convenience methods.
type boundType struct {
	Name    string
	Binder  string
	Methods []string
}

// group is a domain method group.
type group struct {
	Name    string
	Desc    string
	Methods []method
}

func main() {
	if _, err := os.Stat(tlSchema); err != nil {
		fmt.Fprintf(os.Stderr, "apicoverage: run from the repository root: %v\n", err)
		os.Exit(1)
	}

	tlFuncs := parseTLFunctions(tlSchema)
	groups := scanGroups(telegramDir, tlFuncs)
	usedRPC := collectUsedRPC(telegramDir, tlFuncs)
	bound := scanBoundTypes(typesDir)
	paramStructs := scanParams(paramsDir)
	layer := parseTLLayer(tgDir)

	emit(outFile, layer, tlFuncs, usedRPC, groups, bound, paramStructs)
	emitReference("docs/API_REFERENCE.md", layer, groups, bound, paramStructs)
	fmt.Printf("wrote %s and docs/API_REFERENCE.md: %d groups, %d methods, %d bound types, %d params structs, %d/%d TL functions wrapped\n",
		outFile, len(groups), countMethods(groups), len(bound), len(paramStructs), len(usedRPC), len(tlFuncs))
}

func countMethods(groups []group) int {
	n := 0
	for _, g := range groups {
		n += len(g.Methods)
	}
	return n
}

// parseTLFunctions returns the set of Go-visible RPC method names from the
// functions section of a .tl schema (messages.sendMedia → MessagesSendMedia).
func parseTLFunctions(path string) map[string]string {
	data, err := os.ReadFile(path)
	if err != nil {
		panic(err)
	}
	funcs := make(map[string]string) // Go name → TL name
	inFunctions := false
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "---functions---" {
			inFunctions = true
			continue
		}
		if !inFunctions || line == "" || strings.HasPrefix(line, "//") || strings.HasPrefix(line, "---") {
			continue
		}
		name := line[:strings.IndexByte(line, '#')]
		if i := strings.IndexByte(name, '.'); i >= 0 {
			parts := strings.Split(name, ".")
			camel := make([]string, 0, len(parts))
			for _, p := range parts {
				camel = append(camel, strings.ToUpper(p[:1])+p[1:])
			}
			funcs[strings.Join(camel, "")] = name
		} else {
			funcs[strings.ToUpper(name[:1])+name[1:]] = name
		}
	}
	return funcs
}

// parseTLLayer reads the pinned layer number from the generated tg package.
func parseTLLayer(dir string) string {
	matches, _ := filepath.Glob(filepath.Join(dir, "layer_gen.go"))
	if len(matches) == 0 {
		return "?"
	}
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, matches[0], nil, 0)
	if err != nil {
		return "?"
	}
	layer := "?"
	ast.Inspect(f, func(n ast.Node) bool {
		vs, ok := n.(*ast.ValueSpec)
		if !ok || len(vs.Names) == 0 || vs.Names[0].Name != "Layer" {
			return true
		}
		if lit, ok := vs.Values[0].(*ast.BasicLit); ok {
			layer = lit.Value
		}
		return false
	})
	return layer
}

// scanGroups collects the domain method groups with their operations and the
// first sentence of each doc comment.
func scanGroups(dir string, tlFuncs map[string]string) []group {
	// groupDescriptions mirrors telegram/groups.go; the group set itself is
	// validated by the promoted-method collision test, so a missing entry
	// here only degrades the report ordering.
	descs := map[string]string{
		"MessageMethods":    "message lifecycle",
		"ChatMethods":       "chat and channel management",
		"UserMethods":       "user-profile operations",
		"ContactsMethods":   "contact book",
		"AccountMethods":    "account-level settings",
		"AuthMethods":       "authorization",
		"PasswordMethods":   "cloud password",
		"BotsMethods":       "bot surface",
		"BusinessMethods":   "business profile and links",
		"PremiumMethods":    "boosts and premium",
		"StoryMethods":      "stories",
		"PaymentMethods":    "payments, stars, gifts",
		"StickerMethods":    "stickers and custom emoji",
		"FolderMethods":     "chat folders",
		"InviteLinkMethods": "invite links and join requests",
		"UtilityMethods":    "dialogs, peers, files, config",
		"AdvancedMethods":   "raw RPC escape hatches",
		"PhoneMethods":      "calls and live streams",
		"WalletMethods":     "TON wallet",
		"SecretChatMethods": "end-to-end secret chats (mtgo extension)",
	}
	byGroup := make(map[string]map[string]method)
	fset := token.NewFileSet()
	files, _ := filepath.Glob(dir + "/*.go")
	for _, path := range files {
		if strings.HasSuffix(path, "_test.go") {
			continue
		}
		f, err := parser.ParseFile(fset, path, nil, parser.ParseComments)
		if err != nil {
			continue
		}
		for _, decl := range f.Decls {
			fd, ok := decl.(*ast.FuncDecl)
			if !ok || fd.Recv == nil || len(fd.Recv.List) == 0 {
				continue
			}
			group := receiverGroup(fd.Recv.List[0].Type)
			if group == "" || !fd.Name.IsExported() {
				continue
			}
			if _, ok := byGroup[group]; !ok {
				byGroup[group] = make(map[string]method)
			}
			byGroup[group][fd.Name.Name] = method{
				Name: fd.Name.Name,
				Doc:  firstSentence(fd.Doc.Text()),
				Sig:  signature(fd),
			}
		}
	}
	names := make([]string, 0, len(byGroup))
	for g := range byGroup {
		names = append(names, g)
	}
	sort.Strings(names)
	groups := make([]group, 0, len(names))
	for _, n := range names {
		ms := byGroup[n]
		list := make([]method, 0, len(ms))
		for _, m := range ms {
			list = append(list, m)
		}
		sort.Slice(list, func(i, j int) bool { return list[i].Name < list[j].Name })
		groups = append(groups, group{Name: n, Desc: descs[n], Methods: list})
	}
	return groups
}

// receiverGroup returns *XxxMethods from a receiver type expression.
func receiverGroup(expr ast.Expr) string {
	star, ok := expr.(*ast.StarExpr)
	if !ok {
		return ""
	}
	id, ok := star.X.(*ast.Ident)
	if !ok || !strings.HasSuffix(id.Name, "Methods") {
		return ""
	}
	return id.Name
}

// collectUsedRPC walks every non-test file in dir and records selector calls
// whose name matches a TL function name (the generated RPC client methods are
// the only such namespaced identifiers).
func collectUsedRPC(dir string, tlFuncs map[string]string) map[string]string {
	used := make(map[string]string)
	fset := token.NewFileSet()
	files, _ := filepath.Glob(dir + "/*.go")
	for _, path := range files {
		if strings.HasSuffix(path, "_test.go") {
			continue
		}
		f, err := parser.ParseFile(fset, path, nil, 0)
		if err != nil {
			continue
		}
		ast.Inspect(f, func(n ast.Node) bool {
			sel, ok := n.(*ast.SelectorExpr)
			if !ok {
				return true
			}
			if tl, ok := tlFuncs[sel.Sel.Name]; ok {
				used[sel.Sel.Name] = tl
			}
			return true
		})
	}
	return used
}

// scanBoundTypes lists telegram/types types with a SetBinder method and
// their exported bound convenience methods (the Name of a Name/NameContext
// pair, plus single-form methods).
func scanBoundTypes(dir string) []boundType {
	byType := make(map[string]map[string]bool)
	binders := make(map[string]string)
	fset := token.NewFileSet()
	files, _ := filepath.Glob(dir + "/*.go")
	for _, path := range files {
		if strings.HasSuffix(path, "_test.go") {
			continue
		}
		f, err := parser.ParseFile(fset, path, nil, 0)
		if err != nil {
			continue
		}
		for _, decl := range f.Decls {
			fd, ok := decl.(*ast.FuncDecl)
			if !ok || fd.Recv == nil || len(fd.Recv.List) == 0 {
				continue
			}
			recv, ok := fd.Recv.List[0].Type.(*ast.Ident)
			if !ok {
				if star, ok := fd.Recv.List[0].Type.(*ast.StarExpr); ok {
					recv, _ = star.X.(*ast.Ident)
				}
			}
			if recv == nil {
				continue
			}
			if fd.Name.Name == "SetBinder" {
				if ft, ok := fd.Type.Params.List[0].Type.(*ast.Ident); ok {
					binders[recv.Name] = ft.Name
				}
				continue
			}
			if fd.Name.IsExported() {
				if byType[recv.Name] == nil {
					byType[recv.Name] = make(map[string]bool)
				}
				byType[recv.Name][fd.Name.Name] = true
			}
		}
	}
	names := make([]string, 0, len(byType))
	for t := range byType {
		if _, bound := binders[t]; bound {
			names = append(names, t)
		}
	}
	sort.Strings(names)
	out := make([]boundType, 0, len(names))
	for _, n := range names {
		set := byType[n]
		methods := make([]string, 0, len(set))
		for m := range set {
			// Canonical list keeps the convenience form of each pair.
			if strings.HasSuffix(m, "Context") {
				continue
			}
			methods = append(methods, m)
		}
		sort.Strings(methods)
		out = append(out, boundType{Name: n, Binder: binders[n], Methods: methods})
	}
	return out
}

// scanParams lists exported option structs in the params package.
func scanParams(dir string) []string {
	seen := make(map[string]bool)
	fset := token.NewFileSet()
	files, _ := filepath.Glob(dir + "/*.go")
	for _, path := range files {
		if strings.HasSuffix(path, "_test.go") {
			continue
		}
		f, err := parser.ParseFile(fset, path, nil, 0)
		if err != nil {
			continue
		}
		for _, decl := range f.Decls {
			gd, ok := decl.(*ast.GenDecl)
			if !ok || gd.Tok != token.TYPE {
				continue
			}
			for _, spec := range gd.Specs {
				ts, ok := spec.(*ast.TypeSpec)
				if !ok || !ts.Name.IsExported() {
					continue
				}
				if _, isStruct := ts.Type.(*ast.StructType); isStruct {
					seen[ts.Name.Name] = true
				}
			}
		}
	}
	names := make([]string, 0, len(seen))
	for n := range seen {
		names = append(names, n)
	}
	sort.Strings(names)
	return names
}

// firstSentence extracts the first sentence of a doc comment.
func firstSentence(text string) string {
	text = strings.TrimSpace(text)
	if text == "" {
		return ""
	}
	s := doc.Synopsis(text)
	if s == "" {
		s = strings.SplitN(text, "\n", 2)[0]
	}
	return s
}

func emit(path, layer string, tlFuncs, used map[string]string, groups []group, bound []boundType, paramStructs []string) {
	var b strings.Builder
	fmt.Fprintf(&b, "# API Coverage\n\n")
	fmt.Fprintf(&b, "<!-- Generated by cmd/apicoverage from the source tree and compiler/source/api.tl.\n")
	fmt.Fprintf(&b, "     Do not edit by hand; regenerate with `go run ./cmd/apicoverage`. -->\n\n")
	fmt.Fprintf(&b, "Generated %s · TL layer %s · %d domain groups · %d client operations · %d bound types · %d params structs\n\n",
		time.Now().UTC().Format("2006-01-02"), layer, len(groups), countMethods(groups), len(bound), len(paramStructs))

	// RPC coverage
	wrapped := make([]string, 0, len(used))
	for _, tl := range used {
		wrapped = append(wrapped, tl)
	}
	sort.Strings(wrapped)
	pct := 0
	if len(tlFuncs) > 0 {
		pct = 100 * len(used) / len(tlFuncs)
	}
	fmt.Fprintf(&b, "## TL function coverage\n\n")
	fmt.Fprintf(&b, "%d of %d schema functions (%d%%) are invoked by high-level operations. The remainder are\n", len(used), len(tlFuncs), pct)
	fmt.Fprintf(&b, "reachable through the raw RPC client (`client.Raw()`); the list below is the deliberately\n")
	fmt.Fprintf(&b, "unwrapped surface, dominated by channel admin plumbing, MTProto-internal namespaces, and\n")
	fmt.Fprintf(&b, "seldom-used read models.\n\n")

	unwrapped := make([]string, 0)
	for goName, tlName := range tlFuncs {
		if _, ok := used[goName]; !ok {
			unwrapped = append(unwrapped, tlName)
		}
	}
	sort.Strings(unwrapped)
	fmt.Fprintf(&b, "<details><summary>Unwrapped TL functions (%d)</summary>\n\n", len(unwrapped))
	fmt.Fprintf(&b, "```\n")
	for i, n := range unwrapped {
		if i > 0 && i%3 == 0 {
			b.WriteString("\n")
		}
		fmt.Fprintf(&b, "%-44s", n)
	}
	fmt.Fprintf(&b, "\n```\n\n</details>\n\n")

	// Groups
	fmt.Fprintf(&b, "## Client operations by domain\n\n")
	for _, g := range groups {
		title := strings.TrimSuffix(g.Name, "Methods")
		fmt.Fprintf(&b, "### %s\n\n", title)
		if g.Desc != "" {
			fmt.Fprintf(&b, "%s\n\n", g.Desc)
		}
		fmt.Fprintf(&b, "| Method | Purpose |\n|---|---|\n")
		for _, m := range g.Methods {
			fmt.Fprintf(&b, "| `%s` | %s |\n", m.Name, m.Doc)
		}
		fmt.Fprintf(&b, "\n")
	}

	// Bound types
	fmt.Fprintf(&b, "## Bound methods by type\n\n")
	fmt.Fprintf(&b, "Types constructed by the client carry a minimal binder so their convenience methods\n")
	fmt.Fprintf(&b, "work without a client reference. Every bound method has a `NameContext` explicit-context\n")
	fmt.Fprintf(&b, "form; only the convenience forms are listed.\n\n")
	for _, t := range bound {
		fmt.Fprintf(&b, "### %s\n\n", t.Name)
		if t.Binder != "" {
			fmt.Fprintf(&b, "Binder: `%s` (%d methods)\n\n", t.Binder, len(t.Methods))
		}
		fmt.Fprintf(&b, "```\n%s\n```\n\n", strings.Join(t.Methods, ", "))
	}

	// Params
	fmt.Fprintf(&b, "## Option structs (telegram/params)\n\n")
	fmt.Fprintf(&b, "```\n%s\n```\n", strings.Join(paramStructs, ", "))

	if err := os.WriteFile(path, []byte(b.String()), 0o644); err != nil {
		panic(err)
	}
}

// signature renders a compact one-line signature for a function declaration.
func signature(fd *ast.FuncDecl) string {
	var params []string
	if fd.Type.Params != nil {
		for _, field := range fd.Type.Params.List {
			names := make([]string, 0, len(field.Names))
			for _, n := range field.Names {
				names = append(names, n.Name)
			}
			if len(names) == 0 {
				params = append(params, exprString(field.Type))
				continue
			}
			params = append(params, strings.Join(names, ", ")+" "+exprString(field.Type))
		}
	}
	var results []string
	if fd.Type.Results != nil {
		for _, field := range fd.Type.Results.List {
			results = append(results, exprString(field.Type))
		}
	}
	sig := fd.Name.Name + "(" + strings.Join(params, ", ") + ")"
	if len(results) > 0 {
		sig += " (" + strings.Join(results, ", ") + ")"
	}
	return sig
}

// exprString renders a type expression compactly.
func exprString(e ast.Expr) string {
	switch v := e.(type) {
	case *ast.Ident:
		return v.Name
	case *ast.StarExpr:
		return "*" + exprString(v.X)
	case *ast.SelectorExpr:
		return exprString(v.X) + "." + v.Sel.Name
	case *ast.ArrayType:
		if v.Len == nil {
			return "[]" + exprString(v.Elt)
		}
		return "[...]" + exprString(v.Elt)
	case *ast.Ellipsis:
		return "..." + exprString(v.Elt)
	case *ast.MapType:
		return "map[" + exprString(v.Key) + "]" + exprString(v.Value)
	case *ast.InterfaceType:
		return "interface{}"
	case *ast.FuncType:
		return "func(...)"
	default:
		return "_"
	}
}

// emitReference writes docs/API_REFERENCE.md: a curated introduction plus
// generated per-domain operation tables.
func emitReference(path, layer string, groups []group, bound []boundType, paramStructs []string) {
	var b strings.Builder
	b.WriteString(`# mtgo API Reference

> mtgo (MTProto Go) is a Go Telegram MTProto client library.
> This reference is generated from the source tree by ` + "`cmd/apicoverage`" + `; do not edit by hand.

- [Architecture](#architecture)
- [Quick start](#quick-start)
- [Client operations by domain](#client-operations-by-domain)
- [Bound methods by type](#bound-methods-by-type)
- [Option structs](#option-structs)

## Architecture

` + "`telegram.Client`" + ` is composed of twenty domain method groups. Every group method is
promoted onto the client itself, so ` + "`client.SendMessage`" + ` and ` + "`client.MessageMethods.SendMessage`" + `
are the same call; a static collision test guarantees no name is ever defined twice.

Updates arrive through handlers (the ` + "`telegram.Handler`" + ` interface, registered with
` + "`AddHandler`" + `, plus middleware) and carry typed payloads (Message, CallbackQuery, WidgetQuery, …).
Payload types constructed by the client carry a minimal binder interface, so their convenience
methods (Reply, Edit, Answer, …) work without holding a client reference — and every
convenience method has a ` + "`NameContext`" + ` explicit-context form.

Low-level needs are served by ` + "`client.Raw()`" + ` (typed raw RPC), ` + "`InvokeJSON`" + `, and the
` + "`tg`" + ` generated types (TL layer ` + layer + `). The full generated inventory — including the
deliberately unwrapped TL-function list — lives in [api-coverage.md](api-coverage.md).

## Quick start

` + "```go" + `
client, _ := telegram.NewClient(apiID, "api-hash", &telegram.Config{...})
_ = client.Connect(30 * time.Second)

// Phone-code sign-in.
sent, _ := client.SendPhoneNumberCode(ctx, "+1234567890")
user, err := client.SignIn(ctx, "+1234567890", sent.PhoneCodeHash, "12345")
if errors.Is(err, telegram.Err2FARequired) {
    user, err = client.CheckPassword(ctx, "cloud-password")
}

// Send media and text; optional parameters live in params structs.
msg, _ := client.SendMessage(ctx, chatID, "hello",
    &params.SendMessage{ReplyToMessageID: 42})
client.SendPhoto(ctx, chatID, &telegram.InputFile{Path: "cat.jpg"}, "caption")

// Bound convenience methods.
_ = msg.Reply("right back")
updated, _ := msg.EditWidgetData(` + "`{\"v\":2}`" + `, "")
` + "```" + `

`)
	b.WriteString("## Client operations by domain\n\n")
	for _, g := range groups {
		title := strings.TrimSuffix(g.Name, "Methods")
		fmt.Fprintf(&b, "### %s\n\n", title)
		if g.Desc != "" {
			fmt.Fprintf(&b, "%s — `*%s`\n\n", g.Desc, g.Name)
		}
		fmt.Fprintf(&b, "| Operation | Signature | Purpose |\n|---|---|---|\n")
		for _, m := range g.Methods {
			fmt.Fprintf(&b, "| `%s` | `%s` | %s |\n", m.Name, m.Sig, m.Doc)
		}
		b.WriteString("\n")
	}

	b.WriteString("## Bound methods by type\n\n")
	b.WriteString("Convenience forms are listed; each also has a `NameContext` form accepting a `context.Context`.\n\n")
	for _, t := range bound {
		fmt.Fprintf(&b, "- **%s** (%s): %s\n", t.Name, t.Binder, strings.Join(t.Methods, ", "))
	}
	b.WriteString("\n## Option structs\n\n")
	b.WriteString("Optional parameters live in `telegram/params` structs passed as trailing variadic arguments\n")
	b.WriteString("(for example `&params.SendMessage{ReplyToMessageID: 42}`); there are no `With*` helpers.\n\n")
	fmt.Fprintf(&b, "```\n%s\n```\n", strings.Join(paramStructs, ", "))

	if err := os.WriteFile(path, []byte(b.String()), 0o644); err != nil {
		panic(err)
	}
}
