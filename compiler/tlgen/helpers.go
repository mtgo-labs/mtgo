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
				Code: fmt.Sprintf("{ var _f uint32; _f, _ = ReadIntErr(r); v.%s = Fields(_f) }", fd.Name),
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

		wl := buildWriteLine(arg, fd, section, baseTypes)
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

func buildWriteLine(arg Arg, fd fieldData, section string, baseTypes map[string]bool) writeLine {
	wl := writeLine{
		FlagName: fd.FlagName,
		FlagBit:  arg.FlagBit,
	}

	if arg.FlagBit >= 0 {
		wl.IsGuarded = true
	}

	wl.Code = writeExpr(arg, fd.GoType, "v")

	return wl
}

func writeExpr(arg Arg, goType string, receiver string) string {
	fieldName := fieldNameFromTL(arg.Name)
	access := fmt.Sprintf("%s.%s", receiver, fieldName)

	isPtr := strings.HasPrefix(goType, "*") && !strings.HasPrefix(goType, "[]")
	deref := access
	if isPtr {
		deref = "*" + access
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
	case "Bool", "true":
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

func readCallReturnsError(tlType string, section string, baseTypes map[string]bool) bool {
	tlType = normalizeVectorType(tlType)
	tlType = stripNamespace(tlType)
	switch tlType {
	case "int", "long", "int128", "int256", "double", "string", "bytes", "Bool", "true", "#":
		return false
	}
	if strings.HasPrefix(tlType, "Vector<") {
		inner := strings.TrimSuffix(strings.TrimPrefix(tlType, "Vector<"), ">")
		switch stripNamespace(inner) {
		case "int", "long", "string", "bytes":
			return false
		default:
			return true
		}
	}
	return true
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
	argType := normalizeVectorType(arg.Type)
	argType = stripNamespace(argType)

	switch argType {
	case "int":
		if arg.FlagBit >= 0 && strings.HasPrefix(goType, "*") {
			return fmt.Sprintf("_tmp := int32(ReadInt(r)); %s = &_tmp", assign)
		}
		return fmt.Sprintf("%s = int32(ReadInt(r))", assign)
	case "long":
		if arg.FlagBit >= 0 && strings.HasPrefix(goType, "*") {
			return fmt.Sprintf("_tmp := ReadLong(r); %s = &_tmp", assign)
		}
		return fmt.Sprintf("%s = ReadLong(r)", assign)
	case "int128":
		if arg.FlagBit >= 0 {
			return fmt.Sprintf("_tmp := ReadInt128(r); %s = &_tmp", assign)
		}
		return fmt.Sprintf("%s = ReadInt128(r)", assign)
	case "int256":
		if arg.FlagBit >= 0 {
			return fmt.Sprintf("_tmp := ReadInt256(r); %s = &_tmp", assign)
		}
		return fmt.Sprintf("%s = ReadInt256(r)", assign)
	case "double":
		if arg.FlagBit >= 0 && strings.HasPrefix(goType, "*") {
			return fmt.Sprintf("_tmp := ReadDouble(r); %s = &_tmp", assign)
		}
		return fmt.Sprintf("%s = ReadDouble(r)", assign)
	case "string":
		if arg.FlagBit >= 0 && strings.HasPrefix(goType, "*") {
			return fmt.Sprintf("_tmp := ReadString(r); %s = &_tmp", assign)
		}
		return fmt.Sprintf("%s = ReadString(r)", assign)
	case "bytes":
		if arg.FlagBit >= 0 {
			return fmt.Sprintf("%s = ReadBytes(r)", assign)
		}
		return fmt.Sprintf("%s = ReadBytes(r)", assign)
	case "Bool":
		return fmt.Sprintf("%s = ReadBool(r)", assign)
	case "true":
		return fmt.Sprintf("%s = ReadBool(r)", assign)
	case "#":
		return fmt.Sprintf("%s = ReadInt(r)", assign)
	}

	if strings.HasPrefix(argType, "Vector<") {
		inner := strings.TrimSuffix(strings.TrimPrefix(argType, "Vector<"), ">")
		inner = stripNamespace(inner)
		switch inner {
		case "int":
			return fmt.Sprintf("%s = ReadVectorInt(r)", assign)
		case "long":
			return fmt.Sprintf("%s = ReadVectorLong(r)", assign)
		case "string":
			return fmt.Sprintf("%s = ReadVectorString(r)", assign)
		case "bytes":
			return fmt.Sprintf("%s = ReadVectorBytes(r)", assign)
		default:
			elemType := strings.TrimPrefix(goType, "[]")
			idx := fieldNameFromTL(arg.Name)
			return fmt.Sprintf("ReadInt(r); _cnt%s := ReadInt(r); %s = make(%s, _cnt%s); for _i%s := range %s { _obj%s, _ := ReadTLObject(r); %s[_i%s] = %s }", idx, assign, goType, idx, idx, assign, idx, assign, idx, typeAssertExpr("_obj"+idx, elemType))
		}
	}

	return fmt.Sprintf("_obj%s, _ := ReadTLObject(r); %s = %s", fieldNameFromTL(arg.Name), assign, typeAssertExpr("_obj"+fieldNameFromTL(arg.Name), goType))
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
		return "int32(ReadInt(r))"
	case "long":
		return "ReadLong(r)"
	case "int128":
		return "ReadInt128(r)"
	case "int256":
		return "ReadInt256(r)"
	case "double":
		return "ReadDouble(r)"
	case "string":
		return "ReadString(r)"
	case "bytes":
		return "ReadBytes(r)"
	case "Bool", "true":
		return "ReadBool(r)"
	}

	if strings.HasPrefix(tlType, "Vector<") {
		inner := strings.TrimSuffix(strings.TrimPrefix(tlType, "Vector<"), ">")
		switch inner {
		case "int":
			return "ReadVectorInt(r)"
		case "long":
			return "ReadVectorLong(r)"
		case "string":
			return "ReadVectorString(r)"
		case "bytes":
			return "ReadVectorBytes(r)"
		default:
			return "ReadTLObject(r)"
		}
	}

	return "ReadTLObject(r)"
}
