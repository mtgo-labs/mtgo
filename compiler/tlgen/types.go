package tlgen

type Section int

const (
	SectionTypes Section = iota
	SectionFunctions
)

type Arg struct {
	Name     string
	Type     string
	FlagBit  int
	FlagName string
	Generic  bool
}

type Combinator struct {
	Section   Section
	QualName  string
	Namespace string
	Name      string
	ID        uint32
	Args      []Arg
	QualType  string
	TypeSpace string
	Type      string
	Category  string
	IsBare    bool
}

func (c *Combinator) HasFlags() bool {
	for _, a := range c.Args {
		if a.Type == "#" {
			return true
		}
	}
	return false
}

func (c *Combinator) FlagArgs() []Arg {
	var flags []Arg
	for _, a := range c.Args {
		if a.FlagBit >= 0 {
			flags = append(flags, a)
		}
	}
	return flags
}

func (c *Combinator) NonFlagArgs() []Arg {
	var args []Arg
	for _, a := range c.Args {
		if a.FlagBit < 0 && a.Name != "flags" {
			args = append(args, a)
		}
	}
	return args
}
