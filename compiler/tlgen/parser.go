package tlgen

import (
	"bufio"
	"io"
	"regexp"
	"strconv"
	"strings"
)

var (
	sectionRe    = regexp.MustCompile(`^---(\w+)---$`)
	combinatorRe = regexp.MustCompile(`^(\w[\w.]*)#([0-9a-fA-F]+)\s+(.*)=\s*([^;]+);$`)
	argRe        = regexp.MustCompile(`^(\w[\w]*)\s*:\s*(.+)$`)
	flagsRe      = regexp.MustCompile(`^(flags\d*)\.(\d+)\?(.+)$`)
)

func Parse(r io.Reader) ([]Combinator, error) {
	var combos []Combinator
	section := SectionTypes

	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024)

	for scanner.Scan() {
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
			}
			continue
		}

		if m := combinatorRe.FindStringSubmatch(line); m != nil {
			combo := parseCombinator(section, m[1], m[2], m[3], m[4])
			combos = append(combos, combo)
		}
	}

	return combos, scanner.Err()
}

func parseCombinator(section Section, name, hexID, argsStr, retType string) Combinator {
	id, _ := strconv.ParseUint(hexID, 16, 32)

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

	args := parseArgs(argsStr)

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
	}
}

func parseArgs(s string) []Arg {
	if s == "" {
		return nil
	}

	fields := strings.Fields(s)
	var args []Arg

	for _, field := range fields {
		field = strings.TrimSuffix(field, ",")
		m := argRe.FindStringSubmatch(field)
		if m == nil {
			continue
		}

		name := m[1]
		typeStr := m[2]

		if typeStr == "#" {
			args = append(args, Arg{Name: name, Type: "#", FlagBit: -1})
			continue
		}

		if fm := flagsRe.FindStringSubmatch(typeStr); fm != nil {
			bit, _ := strconv.Atoi(fm[2])
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

	return args
}
