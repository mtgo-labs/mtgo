package tlgen

import (
	"fmt"
	"strings"
	"unicode"
)

type fieldData struct {
	Name     string
	GoType   string
	JSONTag  string
	IsFlags  bool
	FlagName string
	FlagBit  int
}

type flagSync struct {
	Field          string
	FlagName       string
	Bit            int
	IsBool         bool
	IsDirectScalar bool
	GoType         string
}

type writeLine struct {
	Code      string
	IsGuarded bool
	IsFlags   bool
	FlagName  string
	FlagBit   int
}

type readLine struct {
	Code string
}

type typeTemplateData struct {
	Name       string
	QualName   string
	ID         uint32
	Fields     []fieldData
	HasFlags   bool
	FlagSyncs  []flagSync
	WriteLines []writeLine
	ReadLines  []readLine
	IsMulti    bool
}

type returnType struct {
	GoType   string
	IsBool   bool
	IsVector bool
}

func filterCombos(combos []Combinator, section Section) []Combinator {
	var result []Combinator
	for _, c := range combos {
		if c.Section == section {
			result = append(result, c)
		}
	}
	return result
}

func computeBaseTypes(combos []Combinator) map[string]bool {
	m := map[string]bool{}
	typeCounts := map[string]int{}
	for _, c := range combos {
		if c.Section == SectionTypes && c.Type != "" {
			typeCounts[c.Type]++
		}
	}
	for t, n := range typeCounts {
		if n > 1 {
			m[t] = true
		}
	}
	return m
}

func computeTypeToConstructor(combos []Combinator) map[string][]Combinator {
	m := map[string][]Combinator{}
	for _, c := range combos {
		if c.Section == SectionTypes && c.Type != "" {
			m[c.Type] = append(m[c.Type], c)
		}
	}
	return m
}

func computeKnownTypes(combos []Combinator) map[string]bool {
	m := map[string]bool{}
	for _, c := range combos {
		if c.Section == SectionTypes {
			m[c.Type] = true
			m[c.QualName] = true
		}
	}
	return m
}

func SnakeToPascal(s string) string {
	parts := strings.Split(s, ".")
	var result []string
	for _, p := range parts {
		p = CamelToSnake(p)
		words := strings.Split(p, "_")
		for _, w := range words {
			if w == "" {
				continue
			}
			result = append(result, snakeWordToPascal(w))
		}
	}
	return strings.Join(result, "")
}

func snakeWordToPascal(s string) string {
	if strings.HasPrefix(s, "id") && len(s) > len("id") {
		suffix := s[len("id"):]
		allDigits := true
		for _, r := range suffix {
			if r < '0' || r > '9' {
				allDigits = false
				break
			}
		}
		if allDigits {
			return "ID" + suffix
		}
	}

	switch s {
	case "id":
		return "ID"
	case "msg":
		return "Msg"
	case "msgs":
		return "Msgs"
	case "dc":
		return "DC"
	case "ttl":
		return "TTL"
	case "url":
		return "URL"
	case "pts":
		return "PTS"
	case "srp":
		return "SRP"
	case "hash":
		return "Hash"
	case "ipv4":
		return "IPv4"
	case "ipv6":
		return "IPv6"
	case "tl":
		return "TL"
	case "mtproto":
		return "MTProto"
	case "rpc":
		return "RPC"
	case "json":
		return "JSON"
	case "api":
		return "API"
	case "tcp":
		return "TCP"
	case "udp":
		return "UDP"
	case "http":
		return "HTTP"
	case "https":
		return "HTTPS"
	case "cdn":
		return "CDN"
	case "rsa":
		return "RSA"
	case "sha":
		return "SHA"
	case "aes":
		return "AES"
	case "sha256":
		return "SHA256"
	case "sha1":
		return "SHA1"
	case "md5":
		return "MD5"
	case "png":
		return "PNG"
	case "jpg", "jpeg":
		return "JPEG"
	case "gif":
		return "GIF"
	case "mp4":
		return "MP4"
	case "ogg":
		return "OGG"
	case "pdf":
		return "PDF"
	case "svg":
		return "SVG"
	case "html":
		return "HTML"
	case "css":
		return "CSS"
	case "sdk":
		return "SDK"
	case "tts":
		return "TTS"
	case "stt":
		return "STT"
	case "uri":
		return "URI"
	case "utc":
		return "UTC"
	case "gmt":
		return "GMT"
	case "ssl":
		return "SSL"
	case "dh":
		return "DH"
	case "pq":
		return "PQ"
	case "ga", "gb", "gp":
		return strings.ToUpper(s)
	case "p", "q":
		return strings.ToUpper(s)
	case "g":
		return "G"
	}

	runes := []rune(s)
	if len(runes) == 1 {
		return strings.ToUpper(s)
	}
	return strings.ToUpper(string(runes[0])) + string(runes[1:])
}

func CamelToSnake(s string) string {
	var result []rune
	runes := []rune(s)
	for i, r := range runes {
		if unicode.IsUpper(r) {
			if i > 0 {
				prev := runes[i-1]
				if unicode.IsLower(prev) || (unicode.IsUpper(prev) && i+1 < len(runes) && unicode.IsLower(runes[i+1])) {
					result = append(result, '_')
				}
			}
			result = append(result, unicode.ToLower(r))
		} else {
			result = append(result, r)
		}
	}
	return string(result)
}

func normalizeVectorType(tlType string) string {
	if strings.HasPrefix(tlType, "vector<") {
		return "Vector" + tlType[len("vector"):]
	}
	return tlType
}

func stripNamespace(tlType string) string {
	if idx := strings.LastIndex(tlType, "."); idx >= 0 {
		if strings.HasPrefix(tlType, "Vector<") {
			return tlType
		}
		return tlType[idx+1:]
	}
	return tlType
}

func goType(tlType string, section string, baseTypes map[string]bool, typeToConstructor map[string][]Combinator) string {
	tlType = normalizeVectorType(tlType)
	tlType = stripNamespace(tlType)
	if strings.HasPrefix(tlType, "Vector<") {
		inner := strings.TrimSuffix(strings.TrimPrefix(tlType, "Vector<"), ">")
		innerGo := goType(inner, section, baseTypes, typeToConstructor)
		return "[]" + innerGo
	}

	if strings.HasPrefix(tlType, "!") {
		return "TLObject"
	}

	switch tlType {
	case "int":
		return "int32"
	case "long":
		return "int64"
	case "int128":
		return "[16]byte"
	case "int256":
		return "[32]byte"
	case "double":
		return "float64"
	case "string":
		return "string"
	case "bytes":
		return "[]byte"
	case "Bool", "true":
		return "bool"
	case "#":
		return "uint32"
	case "Type", "Object":
		return "TLObject"
	}

	if baseTypes != nil && baseTypes[tlType] {
		return SnakeToPascal(tlType) + "Class"
	}

	constructors, ok := typeToConstructor[tlType]
	if ok && len(constructors) == 1 {
		return "*" + SnakeToPascal(constructors[0].QualName)
	}

	if ok && len(constructors) > 1 {
		return SnakeToPascal(tlType) + "Class"
	}

	return "*" + SnakeToPascal(tlType)
}

func resolveGoType(tlType string, baseTypes map[string]bool, typeToConstructor map[string][]Combinator) string {
	return goType(tlType, "types", baseTypes, typeToConstructor)
}

func resolveReturnType(c Combinator, baseTypes map[string]bool, typeToConstructor map[string][]Combinator, knownTypes map[string]bool) returnType {
	tlType := c.Type
	if tlType == "" {
		tlType = c.QualType
	}

	rt := returnType{}

	if strings.HasPrefix(tlType, "Vector<") {
		rt.IsVector = true
		inner := strings.TrimSuffix(strings.TrimPrefix(tlType, "Vector<"), ">")
		if inner == "Bool" || inner == "true" {
			rt.IsBool = false
			rt.GoType = "TLObject"
		} else {
			rt.GoType = "TLObject"
		}
		return rt
	}

	switch tlType {
	case "Bool":
		rt.GoType = "bool"
		rt.IsBool = true
		return rt
	case "true":
		rt.GoType = "bool"
		rt.IsBool = true
		return rt
	case "int":
		rt.GoType = "int32"
		return rt
	case "long":
		rt.GoType = "int64"
		return rt
	case "double":
		rt.GoType = "float64"
		return rt
	case "string":
		rt.GoType = "string"
		return rt
	case "bytes":
		rt.GoType = "[]byte"
		return rt
	}

	if baseTypes[tlType] {
		rt.GoType = SnakeToPascal(tlType) + "Class"
		return rt
	}

	constructors, ok := typeToConstructor[tlType]
	if ok && len(constructors) == 1 {
		rt.GoType = "*" + SnakeToPascal(constructors[0].QualName)
		return rt
	}

	if ok && len(constructors) > 1 {
		rt.GoType = SnakeToPascal(tlType) + "Class"
		return rt
	}

	rt.GoType = "TLObject"
	return rt
}

func fieldNameFromTL(tlName string) string {
	return SnakeToPascal(tlName)
}

func jsonTagFromTL(tlName string) string {
	return CamelToSnake(SnakeToPascal(tlName))
}

func buildTypeData(c Combinator, section string, baseTypes map[string]bool, typeToConstructor map[string][]Combinator, allNames map[string]bool) typeTemplateData {
	td := typeTemplateData{
		Name:     SnakeToPascal(c.QualName),
		QualName: c.QualName,
		ID:       c.ID,
		HasFlags: c.HasFlags(),
	}

	var flagSyncs []flagSync
	var fields []fieldData
	var writeLines []writeLine
	var readLines []readLine

	for _, arg := range c.Args {
		if arg.Type == "#" {
			fd := fieldData{
				Name:    "Flags",
				GoType:  "Fields",
				JSONTag: "-",
				IsFlags: true,
			}
			if arg.Name != "flags" {
				fd.Name = fieldNameFromTL(arg.Name)
			}
			fd.FlagName = arg.Name
			fields = append(fields, fd)

			writeLines = append(writeLines, writeLine{
				Code:    fmt.Sprintf("WriteInt(b, uint32(v.%s))", fd.Name),
				IsFlags: true,
			})

			readLines = append(readLines, readLine{
				Code: fmt.Sprintf("{ var _f uint32; _f, _ = r.ReadUint32(); v.%s = Fields(_f) }", fd.Name),
			})
			continue
		}

		gt := resolveGoTypeForField(arg, section, baseTypes, typeToConstructor)

		fd := fieldData{
			Name:     fieldNameFromTL(arg.Name),
			GoType:   gt,
			JSONTag:  jsonTagFromTL(arg.Name),
			IsFlags:  false,
			FlagName: arg.FlagName,
			FlagBit:  arg.FlagBit,
		}

		if arg.FlagBit >= 0 {
			for _, a := range c.Args {
				if a.Name == arg.FlagName && a.Type == "#" {
					fd.FlagName = fieldNameFromTL(a.Name)
					if a.Name == "flags" {
						fd.FlagName = "Flags"
					}
					break
				}
			}
		}

		if arg.FlagBit >= 0 {
			flagSyncs = append(flagSyncs, flagSync{
				Field:          fd.Name,
				FlagName:       fd.FlagName,
				Bit:            fd.FlagBit,
				IsBool:         gt == "bool",
				IsDirectScalar: isDirectOptionalScalar(gt),
				GoType:         gt,
			})
		}

		fields = append(fields, fd)

		wl := buildWriteLine(arg, fd, section, baseTypes, typeToConstructor)
		writeLines = append(writeLines, wl)

		rl := buildReadLine(arg, fd, section, baseTypes, typeToConstructor)
		readLines = append(readLines, rl)
	}

	td.Fields = fields
	td.FlagSyncs = flagSyncs
	td.WriteLines = writeLines
	td.ReadLines = readLines

	if allNames != nil {
		allNames[td.Name] = true
	}

	return td
}

func resolveGoTypeForField(arg Arg, section string, baseTypes map[string]bool, typeToConstructor map[string][]Combinator) string {
	if arg.FlagBit >= 0 {
		gt := resolveGoType(arg.Type, baseTypes, typeToConstructor)
		if isDirectOptionalScalar(gt) {
			return gt
		}
		if strings.HasPrefix(gt, "*") || strings.HasPrefix(gt, "[]") {
			return gt
		}
		if strings.HasSuffix(gt, "Class") {
			return gt
		}
		switch gt {
		case "bool", "int32", "int64", "float64", "string", "[]byte", "[16]byte", "[32]byte", "uint32", "TLObject":
			return "*" + gt
		}
		return "*" + gt
	}

	return resolveGoType(arg.Type, baseTypes, typeToConstructor)
}

func isDirectOptionalScalar(goType string) bool {
	switch goType {
	case "bool", "int32", "int64", "float64", "string", "uint32":
		return true
	}
	return false
}

func bareVectorInner(tlType string) (string, bool) {
	if strings.HasPrefix(tlType, "vector<") && strings.HasSuffix(tlType, ">") {
		return strings.TrimSuffix(strings.TrimPrefix(tlType, "vector<"), ">"), true
	}
	return "", false
}

func constructorByQualName(name string, typeToConstructor map[string][]Combinator) (Combinator, bool) {
	if typeToConstructor == nil {
		return Combinator{}, false
	}
	stripped := stripNamespace(name)
	for _, constructors := range typeToConstructor {
		for _, c := range constructors {
			if c.QualName == name || c.QualName == stripped || stripNamespace(c.QualName) == stripped {
				return c, true
			}
		}
	}
	return Combinator{}, false
}

func writeBareVectorExpr(inner, access string, typeToConstructor map[string][]Combinator) string {
	switch stripNamespace(inner) {
	case "int":
		return fmt.Sprintf("WriteInt(b, uint32(len(%s))); for _, _item := range %s { WriteInt(b, uint32(_item)) }", access, access)
	case "long":
		return fmt.Sprintf("WriteInt(b, uint32(len(%s))); for _, _item := range %s { WriteLong(b, _item) }", access, access)
	case "string":
		return fmt.Sprintf("WriteInt(b, uint32(len(%s))); for _, _item := range %s { WriteString(b, _item) }", access, access)
	case "bytes":
		return fmt.Sprintf("WriteInt(b, uint32(len(%s))); for _, _item := range %s { WriteBytes(b, _item) }", access, access)
	}

	if c, ok := constructorByQualName(inner, typeToConstructor); ok {
		var body strings.Builder
		for _, a := range c.Args {
			if a.Type == "#" && a.Name == "flags" {
				body.WriteString("WriteInt(b, uint32(_item.Flags))\n")
				continue
			}
			gt := resolveGoTypeForField(a, "types", nil, typeToConstructor)
			body.WriteString(writeExpr(a, gt, "_item", typeToConstructor))
			body.WriteByte('\n')
		}
		return fmt.Sprintf("WriteInt(b, uint32(len(%s))); for _, _item := range %s {\n%s}", access, access, indentCode(strings.TrimRight(body.String(), "\n"), "\t"))
	}

	return fmt.Sprintf("WriteInt(b, uint32(len(%s))); for _, _item := range %s { EncodeTLObject(b, _item) }", access, access)
}

func readBareVectorExpr(inner, assign, fname, goType string, typeToConstructor map[string][]Combinator) string {
	idx := fname
	elemType := strings.TrimPrefix(goType, "[]")
	var buf strings.Builder
	fmt.Fprintf(&buf, "_cnt%s, _ecnt%s := r.ReadUint32()\n", idx, idx)
	fmt.Fprintf(&buf, "if _ecnt%s != nil { return nil, _ecnt%s }\n", idx, idx)
	fmt.Fprintf(&buf, "if _err%s := checkVectorCount(_cnt%s); _err%s != nil {\n\treturn nil, _err%s\n}\n", idx, idx, idx, idx)
	fmt.Fprintf(&buf, "%s = make(%s, _cnt%s)\n", assign, goType, idx)
	fmt.Fprintf(&buf, "for _i%s := range %s {\n", idx, assign)

	switch stripNamespace(inner) {
	case "int":
		fmt.Fprintf(&buf, "\t_item%s, _err%s := r.ReadInt32()\n", idx, idx)
	case "long":
		fmt.Fprintf(&buf, "\t_item%s, _err%s := r.ReadInt64()\n", idx, idx)
	case "string":
		fmt.Fprintf(&buf, "\t_item%s, _err%s := r.ReadString()\n", idx, idx)
	case "bytes":
		fmt.Fprintf(&buf, "\t_item%s, _err%s := r.ReadBytes()\n", idx, idx)
	default:
		if c, ok := constructorByQualName(inner, typeToConstructor); ok {
			fmt.Fprintf(&buf, "\t_item%s, _err%s := Decode%s(r)\n", idx, idx, SnakeToPascal(c.QualName))
		} else {
			fmt.Fprintf(&buf, "\t_obj%s, _err%s := ReadTLObject(r)\n", idx, idx)
			fmt.Fprintf(&buf, "\tif _err%s != nil {\n\t\treturn nil, _err%s\n\t}\n", idx, idx)
			fmt.Fprintf(&buf, "\t_item%s := %s\n", idx, typeAssertExpr("_obj"+idx, elemType))
			fmt.Fprintf(&buf, "\t%s[_i%s] = _item%s\n", assign, idx, idx)
			buf.WriteString("}")
			return buf.String()
		}
	}

	fmt.Fprintf(&buf, "\tif _err%s != nil {\n\t\treturn nil, _err%s\n\t}\n", idx, idx)
	fmt.Fprintf(&buf, "\t%s[_i%s] = _item%s\n", assign, idx, idx)
	buf.WriteString("}")
	return buf.String()
}

func buildWriteLine(arg Arg, fd fieldData, section string, baseTypes map[string]bool, typeToConstructor map[string][]Combinator) writeLine {
	wl := writeLine{
		FlagName: fd.FlagName,
		FlagBit:  arg.FlagBit,
	}

	if arg.FlagBit >= 0 {
		wl.IsGuarded = true
	}

	wl.Code = writeExpr(arg, fd.GoType, "v", typeToConstructor)

	return wl
}

func writeExpr(arg Arg, goType string, receiver string, typeMaps ...map[string][]Combinator) string {
	fieldName := fieldNameFromTL(arg.Name)
	access := fmt.Sprintf("%s.%s", receiver, fieldName)

	isPtr := strings.HasPrefix(goType, "*") && !strings.HasPrefix(goType, "[]")
	deref := access
	if isPtr {
		deref = "*" + access
	}

	var typeToConstructor map[string][]Combinator
	if len(typeMaps) > 0 {
		typeToConstructor = typeMaps[0]
	}
	if inner, ok := bareVectorInner(arg.Type); ok {
		return writeBareVectorExpr(inner, access, typeToConstructor)
	}

	argType := normalizeVectorType(arg.Type)
	argType = stripNamespace(argType)
	switch argType {
	case "int":
		return fmt.Sprintf("WriteInt(b, uint32(%s))", deref)
	case "long":
		return fmt.Sprintf("WriteLong(b, %s)", deref)
	case "int128":
		return fmt.Sprintf("WriteInt128(b, %s)", deref)
	case "int256":
		return fmt.Sprintf("WriteInt256(b, %s)", deref)
	case "double":
		return fmt.Sprintf("WriteDouble(b, %s)", deref)
	case "string":
		return fmt.Sprintf("WriteString(b, %s)", deref)
	case "bytes":
		return fmt.Sprintf("WriteBytes(b, %s)", deref)
	case "true":
		return ""
	case "Bool":
		return fmt.Sprintf("WriteBool(b, %s)", deref)
	}

	if strings.HasPrefix(argType, "Vector<") {
		inner := strings.TrimSuffix(strings.TrimPrefix(argType, "Vector<"), ">")
		switch stripNamespace(inner) {
		case "int":
			return fmt.Sprintf("WriteVectorInt(b, %s)", access)
		case "long":
			return fmt.Sprintf("WriteVectorLong(b, %s)", access)
		case "string":
			return fmt.Sprintf("WriteVectorString(b, %s)", access)
		case "bytes":
			return fmt.Sprintf("WriteVectorBytes(b, %s)", access)
		default:
			return fmt.Sprintf("WriteInt(b, 0x1cb5c415); WriteInt(b, uint32(len(%s))); for _, _item := range %s { EncodeTLObject(b, _item) }", access, access)
		}
	}

	return fmt.Sprintf("EncodeTLObject(b, %s)", access)
}

func buildReadLine(arg Arg, fd fieldData, section string, baseTypes map[string]bool, typeToConstructor map[string][]Combinator) readLine {
	fieldName := fieldNameFromTL(arg.Name)
	assign := fmt.Sprintf("v.%s", fieldName)
	flagAccess := "v." + fd.FlagName

	if arg.FlagBit >= 0 && arg.Type == "true" {
		return readLine{
			Code: fmt.Sprintf("%s = %s.Has(%d)", assign, flagAccess, arg.FlagBit),
		}
	}

	readCode := buildReadExpr(arg, fd.GoType, baseTypes, typeToConstructor)

	if arg.FlagBit >= 0 {
		return readLine{
			Code: fmt.Sprintf("if %s.Has(%d) { %s }", flagAccess, arg.FlagBit, readCode),
		}
	}

	return readLine{Code: readCode}
}

func buildReadExpr(arg Arg, goType string, baseTypes map[string]bool, typeToConstructor map[string][]Combinator) string {
	assign := fmt.Sprintf("v.%s", fieldNameFromTL(arg.Name))
	fname := fieldNameFromTL(arg.Name)
	if inner, ok := bareVectorInner(arg.Type); ok {
		return readBareVectorExpr(inner, assign, fname, goType, typeToConstructor)
	}

	argType := normalizeVectorType(arg.Type)
	argType = stripNamespace(argType)

	switch argType {
	case "int":
		if arg.FlagBit >= 0 && strings.HasPrefix(goType, "*") {
			return fmt.Sprintf("_r%s, _e%s := r.ReadInt32()\nif _e%s != nil { return nil, _e%s }\n_tmp := _r%s; %s = &_tmp", fname, fname, fname, fname, fname, assign)
		}
		return fmt.Sprintf("_r%s, _e%s := r.ReadInt32()\nif _e%s != nil { return nil, _e%s }\n%s = _r%s", fname, fname, fname, fname, assign, fname)
	case "long":
		if arg.FlagBit >= 0 && strings.HasPrefix(goType, "*") {
			return fmt.Sprintf("_r%s, _e%s := r.ReadInt64()\nif _e%s != nil { return nil, _e%s }\n_tmp := _r%s; %s = &_tmp", fname, fname, fname, fname, fname, assign)
		}
		return fmt.Sprintf("_r%s, _e%s := r.ReadInt64()\nif _e%s != nil { return nil, _e%s }\n%s = _r%s", fname, fname, fname, fname, assign, fname)
	case "int128":
		if arg.FlagBit >= 0 {
			return fmt.Sprintf("_r%s, _e%s := r.ReadInt128()\nif _e%s != nil { return nil, _e%s }\n_tmp := _r%s; %s = &_tmp", fname, fname, fname, fname, fname, assign)
		}
		return fmt.Sprintf("_r%s, _e%s := r.ReadInt128()\nif _e%s != nil { return nil, _e%s }\n%s = _r%s", fname, fname, fname, fname, assign, fname)
	case "int256":
		if arg.FlagBit >= 0 {
			return fmt.Sprintf("_r%s, _e%s := r.ReadInt256()\nif _e%s != nil { return nil, _e%s }\n_tmp := _r%s; %s = &_tmp", fname, fname, fname, fname, fname, assign)
		}
		return fmt.Sprintf("_r%s, _e%s := r.ReadInt256()\nif _e%s != nil { return nil, _e%s }\n%s = _r%s", fname, fname, fname, fname, assign, fname)
	case "double":
		if arg.FlagBit >= 0 && strings.HasPrefix(goType, "*") {
			return fmt.Sprintf("_r%s, _e%s := r.ReadFloat64()\nif _e%s != nil { return nil, _e%s }\n_tmp := _r%s; %s = &_tmp", fname, fname, fname, fname, fname, assign)
		}
		return fmt.Sprintf("_r%s, _e%s := r.ReadFloat64()\nif _e%s != nil { return nil, _e%s }\n%s = _r%s", fname, fname, fname, fname, assign, fname)
	case "string":
		if arg.FlagBit >= 0 && strings.HasPrefix(goType, "*") {
			return fmt.Sprintf("_r%s, _e%s := r.ReadString()\nif _e%s != nil { return nil, _e%s }\n_tmp := _r%s; %s = &_tmp", fname, fname, fname, fname, fname, assign)
		}
		return fmt.Sprintf("_r%s, _e%s := r.ReadString()\nif _e%s != nil { return nil, _e%s }\n%s = _r%s", fname, fname, fname, fname, assign, fname)
	case "bytes":
		return fmt.Sprintf("_r%s, _e%s := r.ReadBytes()\nif _e%s != nil { return nil, _e%s }\n%s = _r%s", fname, fname, fname, fname, assign, fname)
	case "Bool":
		return fmt.Sprintf("_r%s, _e%s := r.ReadBool()\nif _e%s != nil { return nil, _e%s }\n%s = _r%s", fname, fname, fname, fname, assign, fname)
	case "true":
		return fmt.Sprintf("_r%s, _e%s := r.ReadBool()\nif _e%s != nil { return nil, _e%s }\n%s = _r%s", fname, fname, fname, fname, assign, fname)
	case "#":
		return fmt.Sprintf("_r%s, _e%s := r.ReadUint32()\nif _e%s != nil { return nil, _e%s }\n%s = _r%s", fname, fname, fname, fname, assign, fname)
	}

	if strings.HasPrefix(argType, "Vector<") {
		inner := strings.TrimSuffix(strings.TrimPrefix(argType, "Vector<"), ">")
		inner = stripNamespace(inner)
		switch inner {
		case "int":
			return fmt.Sprintf("_vv%s, _ve%s := r.ReadVectorInt()\nif _ve%s != nil { return nil, _ve%s }\n%s = _vv%s", fname, fname, fname, fname, assign, fname)
		case "long":
			return fmt.Sprintf("_vv%s, _ve%s := r.ReadVectorLong()\nif _ve%s != nil { return nil, _ve%s }\n%s = _vv%s", fname, fname, fname, fname, assign, fname)
		case "string":
			return fmt.Sprintf("_vv%s, _ve%s := r.ReadVectorString()\nif _ve%s != nil { return nil, _ve%s }\n%s = _vv%s", fname, fname, fname, fname, assign, fname)
		case "bytes":
			return fmt.Sprintf("_vv%s, _ve%s := r.ReadVectorBytes()\nif _ve%s != nil { return nil, _ve%s }\n%s = _vv%s", fname, fname, fname, fname, assign, fname)
		default:
			elemType := strings.TrimPrefix(goType, "[]")
			idx := fieldNameFromTL(arg.Name)
			var buf strings.Builder
			fmt.Fprintf(&buf, "_vhdr%s, _ehdr%s := r.ReadUint32()\n", idx, idx)
			fmt.Fprintf(&buf, "if _ehdr%s != nil { return nil, _ehdr%s }\n", idx, idx)
			fmt.Fprintf(&buf, "_cnt%s, _ecnt%s := r.ReadUint32()\n", idx, idx)
			fmt.Fprintf(&buf, "if _ecnt%s != nil { return nil, _ecnt%s }\n", idx, idx)
			fmt.Fprintf(&buf, "if _err%s := checkVectorCount(_cnt%s); _err%s != nil {\n\treturn nil, _err%s\n}\n", idx, idx, idx, idx)
			fmt.Fprintf(&buf, "%s = make(%s, _cnt%s)\n", assign, goType, idx)
			fmt.Fprintf(&buf, "for _i%s := range %s {\n", idx, assign)
			fmt.Fprintf(&buf, "\t_obj%s, _err%s := ReadTLObject(r)\n", idx, idx)
			fmt.Fprintf(&buf, "\tif _err%s != nil {\n\t\treturn nil, _err%s\n\t}\n", idx, idx)
			fmt.Fprintf(&buf, "\t%s[_i%s] = %s\n", assign, idx, typeAssertExpr("_obj"+idx, elemType))
			fmt.Fprintf(&buf, "}\n_ = _vhdr%s", idx)
			return buf.String()
		}
	}

	return fmt.Sprintf("_obj%s, _err%s := ReadTLObject(r)\nif _err%s != nil {\n\treturn nil, _err%s\n}\n%s = %s", fname, fname, fname, fname, assign, typeAssertExpr("_obj"+fname, goType))
}

func typeAssertExpr(varName string, goType string) string {
	if strings.HasPrefix(goType, "[]") {
		return fmt.Sprintf("%s(%s)", goType, varName)
	}
	if strings.HasPrefix(goType, "*") && !strings.HasPrefix(goType, "[]") {
		inner := goType[1:]
		if strings.HasSuffix(inner, "Class") {
			return fmt.Sprintf("%s.(%s)", varName, inner)
		}
		return fmt.Sprintf("%s.(%s)", varName, goType)
	}
	if strings.HasSuffix(goType, "Class") {
		return fmt.Sprintf("%s.(%s)", varName, goType)
	}
	return fmt.Sprintf("%s.(%s)", varName, goType)
}

func readFuncName(tlType string, section string, baseTypes map[string]bool) string {
	tlType = normalizeVectorType(tlType)
	switch tlType {
	case "int":
		return "r.ReadInt32()"
	case "long":
		return "r.ReadInt64()"
	case "int128":
		return "r.ReadInt128()"
	case "int256":
		return "r.ReadInt256()"
	case "double":
		return "r.ReadFloat64()"
	case "string":
		return "r.ReadString()"
	case "bytes":
		return "r.ReadBytes()"
	case "Bool", "true":
		return "r.ReadBool()"
	}

	if strings.HasPrefix(tlType, "Vector<") {
		inner := strings.TrimSuffix(strings.TrimPrefix(tlType, "Vector<"), ">")
		switch inner {
		case "int":
			return "r.ReadVectorInt()"
		case "long":
			return "r.ReadVectorLong()"
		case "string":
			return "r.ReadVectorString()"
		case "bytes":
			return "r.ReadVectorBytes()"
		default:
			return "ReadTLObject(r)"
		}
	}

	return "ReadTLObject(r)"
}

func prefixE2EType(goType string) string {
	replacements := []struct{ from, to string }{
		{"Fields", "tg.Fields"},
	}
	for _, r := range replacements {
		goType = strings.ReplaceAll(goType, r.from, r.to)
	}
	return goType
}
