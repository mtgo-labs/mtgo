package telegram

import "github.com/mtgo-labs/mtgo/internal/storage"

func newDefaultStorage(_ string) (storage.Storage, error) {
	return NewMemoryStorage(), nil
}
