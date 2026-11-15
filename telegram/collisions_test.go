package telegram

import (
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"strings"
	"testing"
)

// TestNoPromotedMethodCollisions guarantees that no exported method name is
// defined on more than one method group, and that no group method is shadowed
// by a method defined directly on Client. Go resolves both situations silently
// (ambiguous selectors error only at use time; a direct Client method always
// outranks a promoted one), so this test is the structural gate that keeps the
// promoted-method surface unambiguous as groups evolve.
func TestNoPromotedMethodCollisions(t *testing.T) {
	files, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatal(err)
	}

	fset := token.NewFileSet()
	// method name -> receiver type names that define it (exported methods only)
	owners := map[string]map[string]bool{}

	for _, name := range files {
		if strings.HasSuffix(name, "_test.go") {
			continue
		}
		f, err := parser.ParseFile(fset, name, nil, 0)
		if err != nil {
			t.Fatalf("parse %s: %v", name, err)
		}
		for _, decl := range f.Decls {
			fd, ok := decl.(*ast.FuncDecl)
			if !ok || fd.Recv == nil || len(fd.Recv.List) == 0 || !fd.Name.IsExported() {
				continue
			}
			recvType := receiverTypeName(fd.Recv.List[0].Type)
			if recvType == "" || !isMethodGroup(recvType) {
				continue
			}
			if owners[fd.Name.Name] == nil {
				owners[fd.Name.Name] = map[string]bool{}
			}
			owners[fd.Name.Name][recvType] = true
		}
	}

	for method, defs := range owners {
		if len(defs) > 1 {
			t.Errorf("method %s is defined on multiple groups: %v", method, keys(defs))
		}
		if defs["Client"] && len(defs) > 1 {
			t.Errorf("method %s is defined both on Client and on a group: %v", method, keys(defs))
		}
	}

	// Cross-check against the live promoted method set: every group method
	// must be selectable on Client (embedding present) — verified structurally
	// by asserting each group type is embedded in the Client declaration.
	embedded := embeddedGroups(t, fset)
	for group := range groupTypes {
		if !embedded[group] {
			t.Errorf("group %s is not embedded in Client", group)
		}
	}
}

// groupTypes is the authoritative set of domain method groups. A method whose
// receiver is one of these types participates in promotion onto *Client.
var groupTypes = map[string]bool{
	"AccountMethods":    true,
	"AdvancedMethods":   true,
	"AuthMethods":       true,
	"BotsMethods":       true,
	"BusinessMethods":   true,
	"ChatMethods":       true,
	"ContactsMethods":   true,
	"FolderMethods":     true,
	"InviteLinkMethods": true,
	"MessageMethods":    true,
	"PasswordMethods":   true,
	"PaymentMethods":    true,
	"PhoneMethods":      true,
	"PremiumMethods":    true,
	"SecretChatMethods": true,
	"StickerMethods":    true,
	"StoryMethods":      true,
	"UserMethods":       true,
	"UtilityMethods":    true,
	"WalletMethods":     true,
}

func isMethodGroup(typeName string) bool {
	if typeName == "Client" {
		return true
	}
	return groupTypes[typeName]
}

func receiverTypeName(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.StarExpr:
		if id, ok := t.X.(*ast.Ident); ok {
			return id.Name
		}
	case *ast.Ident:
		return t.Name
	}
	return ""
}

func embeddedGroups(t *testing.T, fset *token.FileSet) map[string]bool {
	t.Helper()
	f, err := parser.ParseFile(fset, "client.go", nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	for _, decl := range f.Decls {
		gd, ok := decl.(*ast.GenDecl)
		if !ok || gd.Tok != token.TYPE {
			continue
		}
		ts, ok := gd.Specs[0].(*ast.TypeSpec)
		if !ok || ts.Name.Name != "Client" {
			continue
		}
		st, ok := ts.Type.(*ast.StructType)
		if !ok {
			t.Fatal("Client is not a struct")
		}
		out := map[string]bool{}
		for _, field := range st.Fields.List {
			if star, ok := field.Type.(*ast.StarExpr); ok {
				if id, ok := star.X.(*ast.Ident); ok {
					out[id.Name] = true
				}
			}
		}
		return out
	}
	t.Fatal("Client type not found")
	return nil
}

func keys(m map[string]bool) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}
