package session

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/mtgo-labs/mtgo/tg"
)

// DcAuthStateType represents the auth state for a non-main DC.
// Ported from td/td/telegram/net/DcAuthManager.h:43-44 (State enum).
type DcAuthStateType int

const (
	// DcAuthWaiting is the initial state — waiting for main DC auth.
	DcAuthWaiting DcAuthStateType = iota
	// DcAuthExport is exporting auth from main DC.
	DcAuthExport
	// DcAuthImport is importing auth to target DC.
	DcAuthImport
	// DcAuthBeforeOk is auth imported, waiting for first successful RPC.
	DcAuthBeforeOk
	// DcAuthOk is auth confirmed by successful RPC.
	DcAuthOk
)

func (s DcAuthStateType) String() string {
	switch s {
	case DcAuthWaiting:
		return "waiting"
	case DcAuthExport:
		return "export"
	case DcAuthImport:
		return "import"
	case DcAuthBeforeOk:
		return "before_ok"
	case DcAuthOk:
		return "ok"
	default:
		return "unknown"
	}
}

// DcAuthState tracks auth state for a single DC.
// Ported from td/td/telegram/net/DcAuthManager.h:38-48 (DcInfo).
type DcAuthState struct {
	DCID        int
	State       DcAuthStateType
	ExportBytes []byte
	ExportID    int64
	WaitID      int64
	Attempts    int
}

// ExportFunc exports auth from one DC to another.
type ExportFunc func(ctx context.Context, fromDC, toDC int) (*tg.AuthExportedAuthorization, error)

// ImportFunc imports auth to a DC. The dcID is provided by the caller context.
type ImportFunc func(ctx context.Context, id int64, b []byte) error

// DcAuthManager manages auth export/import for non-main DCs.
// Ported from td/td/telegram/net/DcAuthManager.h:27-69.
type DcAuthManager struct {
	mainDCID   int
	dcs        map[int]*DcAuthState
	exportFunc ExportFunc
	importers  map[int]ImportFunc
	mu         sync.Mutex
}

// NewDcAuthManager creates a new DC auth manager.
func NewDcAuthManager(mainDCID int, exportFunc ExportFunc, importFunc ImportFunc, _ any) *DcAuthManager {
	return &DcAuthManager{
		mainDCID:   mainDCID,
		dcs:        make(map[int]*DcAuthState),
		exportFunc: exportFunc,
		importers:  make(map[int]ImportFunc),
	}
}

// SetImporter sets or clears the import function for a DC.
func (m *DcAuthManager) SetImporter(dcID int, fn ImportFunc) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if fn == nil {
		delete(m.importers, dcID)
	} else {
		m.importers[dcID] = fn
	}
}

// AddDC registers a new DC for auth management.
func (m *DcAuthManager) AddDC(dcID int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.dcs[dcID]; !ok {
		m.dcs[dcID] = &DcAuthState{
			DCID:  dcID,
			State: DcAuthWaiting,
		}
	}
}

// UpdateMainDC updates the main DC ID.
func (m *DcAuthManager) UpdateMainDC(newMainDCID int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.mainDCID = newMainDCID
}

// IsAuthorized reports whether the DC has auth.
func (m *DcAuthManager) IsAuthorized(dcID int) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	dc, ok := m.dcs[dcID]
	if !ok {
		return false
	}
	return dc.State == DcAuthOk || dc.State == DcAuthBeforeOk
}

// ExportID returns the export ID for a DC.
func (m *DcAuthManager) ExportID(dcID int) int64 {
	m.mu.Lock()
	defer m.mu.Unlock()
	dc, ok := m.dcs[dcID]
	if !ok {
		return 0
	}
	return dc.ExportID
}

// GetState returns the auth state for a DC.
func (m *DcAuthManager) GetState(dcID int) DcAuthStateType {
	m.mu.Lock()
	defer m.mu.Unlock()
	dc, ok := m.dcs[dcID]
	if !ok {
		return DcAuthWaiting
	}
	return dc.State
}

// SetState sets the auth state for a DC.
func (m *DcAuthManager) SetState(dcID int, state DcAuthStateType) {
	m.mu.Lock()
	defer m.mu.Unlock()
	dc, ok := m.dcs[dcID]
	if !ok {
		return
	}
	dc.State = state
}

// DCLoop drives the state machine for a DC.
// Ported from td/td/telegram/net/DcAuthManager.cpp (dc_loop).
func (m *DcAuthManager) DCLoop(ctx context.Context, dcID int) error {
	// Auto-register DC if not already tracked.
	m.mu.Lock()
	dc, ok := m.dcs[dcID]
	if !ok {
		dc = &DcAuthState{
			DCID:  dcID,
			State: DcAuthWaiting,
		}
		m.dcs[dcID] = dc
	}
	m.mu.Unlock()

	for {
		m.mu.Lock()
		state := dc.State
		m.mu.Unlock()

		switch state {
		case DcAuthWaiting:
			// Export auth from main DC.
			result, err := m.exportFunc(ctx, m.mainDCID, dcID)
			if err != nil {
				m.mu.Lock()
				dc.Attempts++
				m.mu.Unlock()
				if dc.Attempts >= 3 {
					return fmt.Errorf("dc_auth: export failed after 3 attempts: %w", err)
				}
				time.Sleep(time.Duration(dc.Attempts) * time.Second)
				continue
			}

			m.mu.Lock()
			dc.ExportBytes = result.Bytes
			dc.ExportID = result.ID
			dc.State = DcAuthImport
			m.mu.Unlock()

		case DcAuthExport:
			// Wait for export to complete.
			time.Sleep(100 * time.Millisecond)

		case DcAuthImport:
			// Import auth to target DC.
			m.mu.Lock()
			authBytes := dc.ExportBytes
			exportID := dc.ExportID
			importer := m.importers[dcID]
			m.mu.Unlock()

			if importer == nil {
				return fmt.Errorf("dc_auth: no importer for DC %d", dcID)
			}

			err := importer(ctx, exportID, authBytes)
			if err != nil {
				m.mu.Lock()
				dc.Attempts++
				m.mu.Unlock()
				if dc.Attempts >= 3 {
					return fmt.Errorf("dc_auth: import failed after 3 attempts: %w", err)
				}
				time.Sleep(time.Duration(dc.Attempts) * time.Second)
				continue
			}

			m.mu.Lock()
			dc.State = DcAuthBeforeOk
			m.mu.Unlock()

		case DcAuthBeforeOk:
			// Auth imported, waiting for first successful RPC.
			m.mu.Lock()
			dc.State = DcAuthOk
			m.mu.Unlock()
			return nil

		case DcAuthOk:
			// Auth confirmed.
			return nil
		}
	}
}

// Cleanup removes all DC auth states.
func (m *DcAuthManager) Cleanup() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.dcs = make(map[int]*DcAuthState)
}
