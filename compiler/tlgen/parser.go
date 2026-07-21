package tlgen

import (
	"bufio"
	"fmt"
	"hash/crc32"
	"io"
	"regexp"
	"strconv"
	"strings"
)

var (
	sectionRe    = regexp.MustCompile(`^---(\w+)---$`)
	combinatorRe = regexp.MustCompile(`^(\w[\w.]*)(?:#([0-9a-fA-F]+))?(?:\s+(.*?))?\s*=\s*([^;]+);$`)
	argRe        = regexp.MustCompile(`^(\w[\w]*)\s*:\s*(.+)$`)
	flagsRe      = regexp.MustCompile(`^(flags\d*)\.(\d+)\?(.+)$`)
)

var builtinDeclarations = map[string]struct{}{
	"int ? = Int;":                                 {},
	"long ? = Long;":                               {},
	"double ? = Double;":                           {},
	"string ? = String;":                           {},
	"int128 4*[ int ] = Int128;":                   {},
	"int256 8*[ int ] = Int256;":                   {},
	"vector {t:Type} # [ t ] = Vector t;":          {},
	"vector#1cb5c415 {t:Type} # [ t ] = Vector t;": {},
}

func Parse(r io.Reader) ([]Combinator, error) {
	var combos []Combinator
	section := SectionTypes
	lineNo := 0

	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024)

	for scanner.Scan() {
		lineNo++
		line := strings.TrimSpace(scanner.Text())

		if line == "" || strings.HasPrefix(line, "//") {
			continue
		}

		if m := sectionRe.FindStringSubmatch(line); m != nil {
			switch m[1] {
			case "types":
				section = SectionTypes
			case "functions":
				section = SectionFunctions
			default:
				return nil, fmt.Errorf("line %d: unknown section %q", lineNo, m[1])
			}
			continue
		}

		if _, ok := builtinDeclarations[line]; ok {
			continue
		}

		m := combinatorRe.FindStringSubmatch(line)
		if m == nil {
			return nil, fmt.Errorf("line %d: unsupported TL declaration %q", lineNo, line)
		}

		combo, err := parseCombinator(section, m[1], m[2], m[3], m[4], strings.TrimSuffix(line, ";"))
		if err != nil {
			return nil, fmt.Errorf("line %d: %w", lineNo, err)
		}
		combos = append(combos, combo)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("line %d: %w", lineNo, err)
	}
	return combos, nil
}

func parseCombinator(section Section, name, hexID, argsStr, retType, declaration string) (Combinator, error) {
	var id uint64
	if hexID == "" {
		id = uint64(crc32.ChecksumIEEE([]byte(declaration)))
	} else {
		parsed, err := strconv.ParseUint(hexID, 16, 32)
		if err != nil {
			return Combinator{}, fmt.Errorf("invalid constructor id %q: %w", hexID, err)
		}
		id = parsed
	}

	parts := strings.SplitN(name, ".", 2)
	namespace := ""
	cleanName := name
	if len(parts) == 2 {
		namespace = parts[0]
		cleanName = parts[1]
	}

	typeParts := strings.SplitN(retType, ".", 2)
	typeSpace := ""
	typeName := retType
	if len(typeParts) == 2 {
		typeSpace = typeParts[0]
		typeName = typeParts[1]
	}

	args, err := parseArgs(argsStr)
	if err != nil {
		return Combinator{}, err
	}

	category := "Types"
	if section == SectionFunctions {
		category = "Functions"
	}

	return Combinator{
		Section:   section,
		QualName:  name,
		Namespace: namespace,
		Name:      cleanName,
		ID:        uint32(id),
		Args:      args,
		QualType:  retType,
		TypeSpace: typeSpace,
		Type:      typeName,
		Category:  category,
	}, nil
}

func parseArgs(s string) ([]Arg, error) {
	if s == "" {
		return nil, nil
	}

	fields := strings.Fields(s)
	var args []Arg

	for _, field := range fields {
		field = strings.TrimSuffix(field, ",")
		if field == "" {
			continue
		}

		// Skip generic type parameters, e.g. {X:Type}.
		if strings.HasPrefix(field, "{") {
			continue
		}

		m := argRe.FindStringSubmatch(field)
		if m == nil {
			return nil, fmt.Errorf("unsupported argument token %q", field)
		}

		name := m[1]
		typeStr := m[2]

		if typeStr == "#" {
			args = append(args, Arg{Name: name, Type: "#", FlagBit: -1})
			continue
		}

		if fm := flagsRe.FindStringSubmatch(typeStr); fm != nil {
			bit, err := strconv.Atoi(fm[2])
			if err != nil {
				return nil, fmt.Errorf("invalid flag bit %q in field %q: %w", fm[2], field, err)
			}
			if bit >= 32 {
				return nil, fmt.Errorf("invalid flag bit %d in field %q: must be between 0 and 31", bit, field)
			}
			args = append(args, Arg{
				Name:     name,
				Type:     fm[3],
				FlagBit:  bit,
				FlagName: fm[1],
			})
			continue
		}

		isGeneric := false
		if strings.HasPrefix(typeStr, "{") {
			isGeneric = true
		}

		args = append(args, Arg{
			Name:    name,
			Type:    typeStr,
			FlagBit: -1,
			Generic: isGeneric,
		})
	}

	return args, nil
}
