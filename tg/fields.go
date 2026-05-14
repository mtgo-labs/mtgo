package tg

type Fields uint32

func (f Fields) Has(n int) bool {
	return f&(1<<n) != 0
}

func (f *Fields) Set(n int) {
	*f |= 1 << n
}

func (f *Fields) Unset(n int) {
	*f &= ^(1 << n)
}
