package telegram

import "github.com/mtgo-labs/mtgo/telegram/params"

func getOptDef[T comparable](def T, opts ...T) T {
	return params.GetOptDef(def, opts...)
}
