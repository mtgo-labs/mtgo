package session

import (
	"context"
	"errors"
	"testing"

	"github.com/mtgo-labs/mtgo/tg"
)

func TestDcAuthManager(t *testing.T) {
	var exports, imports int
	mgr := NewDcAuthManager(
		2,
		func(ctx context.Context, fromDC, toDC int) (*tg.AuthExportedAuthorization, error) {
			exports++
			if fromDC != 2 || toDC != 4 {
				t.Fatalf("export from/to = %d/%d, want 2/4", fromDC, toDC)
			}
			return &tg.AuthExportedAuthorization{ID: 42, Bytes: []byte("auth")}, nil
		},
		nil,
		nil,
	)

	// Set importer for DC 4 before calling DCLoop.
	mgr.SetImporter(4, func(ctx context.Context, id int64, b []byte) error {
		imports++
		if id != 42 || string(b) != "auth" {
			t.Fatalf("import id/bytes = %d/%q, want 42/auth", id, b)
		}
		return nil
	})

	if err := mgr.DCLoop(context.Background(), 4); err != nil {
		t.Fatalf("DCLoop() error: %v", err)
	}
	if exports != 1 || imports != 1 {
		t.Fatalf("exports/imports = %d/%d, want 1/1", exports, imports)
	}
	if !mgr.IsAuthorized(4) {
		t.Fatal("dc 4 is not authorized")
	}
	if got := mgr.ExportID(4); got != 42 {
		t.Fatalf("ExportID() = %d, want 42", got)
	}
}

func TestDcAuthManagerRetry(t *testing.T) {
	var exports int
	mgr := NewDcAuthManager(
		2,
		func(ctx context.Context, fromDC, toDC int) (*tg.AuthExportedAuthorization, error) {
			exports++
			if exports == 1 {
				return nil, errors.New("temporary")
			}
			return &tg.AuthExportedAuthorization{ID: 7, Bytes: []byte("ok")}, nil
		},
		nil,
		nil,
	)

	mgr.SetImporter(4, func(ctx context.Context, id int64, b []byte) error {
		return nil
	})

	if err := mgr.DCLoop(context.Background(), 4); err != nil {
		t.Fatalf("DCLoop() error: %v", err)
	}
	if exports != 2 {
		t.Fatalf("exports = %d, want 2", exports)
	}
}
