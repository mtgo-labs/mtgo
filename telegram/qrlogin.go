package telegram

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/mtgo-labs/mtgo/telegram/types"
	"github.com/mtgo-labs/mtgo/tg"
)

// QRLoginToken holds the raw token bytes and expiration time for a QR code login session.
//
// Example:
//
//	token, err := client.GetQRCodeLoginToken(ctx)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	fmt.Printf("Token expires at %d\n", token.Expires)
type QRLoginToken struct {
	// Token is the raw QR code token bytes to be encoded into a QR code image.
	Token []byte
	// Expires is the Unix timestamp when the token expires and a new one must be
	// requested.
	Expires int32
}

// GetQRCodeLoginToken requests a new QR code login token from Telegram. The returned
// QRLoginToken can be encoded into a QR code for scanning by an already-authenticated
// Telegram client. Returns an error if the client is not connected or if DC migration
// is required (not yet supported).
//
// Example:
//
//	ctx := context.Background()
//	token, err := client.GetQRCodeLoginToken(ctx)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	fmt.Printf("QR token expires at unix %d\n", token.Expires)
func (c *Client) GetQRCodeLoginToken(ctx context.Context) (*QRLoginToken, error) {
	if err := c.ensureConnected(); err != nil {
		return nil, err
	}

	c.Log.Debug("auth: requesting QR login token")
	rpc := c.Raw()
	result, err := rpc.AuthExportLoginToken(ctx, &tg.AuthExportLoginTokenRequest{
		APIID:     c.cfg.APIID,
		APIHash:   c.cfg.APIHash,
		ExceptIds: nil,
	})
	if err != nil {
		c.Log.Errorf("auth: QR login token request failed: %v", err)
		return nil, fmt.Errorf("export login token: %w", err)
	}

	switch v := result.(type) {
	case *tg.AuthLoginToken:
		c.Log.Debugf("auth: QR login token received expires=%d", v.Expires)
		return &QRLoginToken{
			Token:   v.Token,
			Expires: v.Expires,
		}, nil
	case *tg.AuthLoginTokenMigrateTo:
		return c.qrLoginMigrateToDC(ctx, int(v.DCID))
	default:
		return nil, fmt.Errorf("unexpected login token type %T", result)
	}
}

// CheckQRCodeLoginToken checks whether the given token has been scanned by an
// authenticated Telegram client. On success it returns the authorized User. Returns
// an error if the QR code has not yet been scanned, if DC migration is required, or
// if the token is invalid or expired.
//
// Example:
//
//	ctx := context.Background()
//	user, err := client.CheckQRCodeLoginToken(ctx, token.Token)
//	if err != nil {
//	    log.Printf("Not scanned yet: %v", err)
//	    return
//	}
//	fmt.Printf("Logged in as %s\n", user.FirstName)
func (c *Client) CheckQRCodeLoginToken(ctx context.Context, token []byte) (*types.User, error) {
	if err := c.ensureConnected(); err != nil {
		return nil, err
	}

	c.Log.Debug("auth: checking QR login token")
	rpc := c.Raw()
	result, err := rpc.AuthImportLoginToken(ctx, &tg.AuthImportLoginTokenRequest{
		Token: token,
	})
	if err != nil {
		c.Log.Errorf("auth: QR login token check failed: %v", err)
		return nil, fmt.Errorf("import login token: %w", err)
	}

	switch v := result.(type) {
	case *tg.AuthLoginTokenSuccess:
		auth, ok := v.Authorization.(*tg.AuthAuthorization)
		if !ok {
			return nil, fmt.Errorf("unexpected authorization type %T", v.Authorization)
		}
		c.Log.Info("auth: QR login successful")
		user := types.ParseUser(auth.User)
		c.SetMe(user)
		c.saveMeToStorage(user)
		return user, nil
	case *tg.AuthLoginToken:
		return nil, fmt.Errorf("QR code not yet scanned, retry later (expires in %d seconds)", v.Expires)
	case *tg.AuthLoginTokenMigrateTo:
		return nil, fmt.Errorf("QR login token requires DC migration to DC %d", v.DCID)
	default:
		return nil, fmt.Errorf("unexpected login token type %T", result)
	}
}

// qrLoginMigrateToDC handles the AuthLoginTokenMigrateTo response by
// exporting authorization to the target DC, re-connecting, and re-issuing
// the export login token request on the correct DC.
func (c *Client) qrLoginMigrateToDC(ctx context.Context, targetDC int) (*QRLoginToken, error) {
	c.Log.Infof("auth: QR login migrating to DC %d", targetDC)

	st := c.storage
	if st == nil {
		return nil, fmt.Errorf("QR login DC migration: no storage available")
	}

	rpc, err := c.dcRPC(ctx, targetDC)
	if err != nil {
		return nil, fmt.Errorf("QR login DC migration: dc rpc: %w", err)
	}

	exported, err := rpc.AuthExportAuthorization(ctx, &tg.AuthExportAuthorizationRequest{DCID: int32(targetDC)})
	if err != nil {
		return nil, fmt.Errorf("export authorization for DC %d: %w", targetDC, err)
	}

	c.cleanupSessions(false)

	c.migratingDC.Store(true)

	if err := st.SetDCID(targetDC); err != nil {
		return nil, fmt.Errorf("save dc_id: %w", err)
	}

	if err := c.connectTransport(30 * time.Second); err != nil {
		return nil, fmt.Errorf("reconnect to DC %d: %w", targetDC, err)
	}

	authImport := &tg.AuthImportAuthorizationRequest{
		ID:    exported.ID,
		Bytes: exported.Bytes,
	}
	if _, err := c.Raw().AuthImportAuthorization(ctx, authImport); err != nil {
		c.Log.Warnf("auth: import authorization on DC %d failed (may already be authorized): %v", targetDC, err)
	}

	result, err := c.Raw().AuthExportLoginToken(ctx, &tg.AuthExportLoginTokenRequest{
		APIID:     c.cfg.APIID,
		APIHash:   c.cfg.APIHash,
		ExceptIds: nil,
	})
	if err != nil {
		return nil, fmt.Errorf("export login token on DC %d: %w", targetDC, err)
	}

	switch v := result.(type) {
	case *tg.AuthLoginToken:
		c.Log.Debugf("auth: QR login token received on DC %d expires=%d", targetDC, v.Expires)
		fmt.Fprintf(os.Stderr, "QR login: migrated to DC %d, token expires in %d seconds\n", targetDC, v.Expires)
		return &QRLoginToken{
			Token:   v.Token,
			Expires: v.Expires,
		}, nil
	case *tg.AuthLoginTokenMigrateTo:
		return nil, fmt.Errorf("QR login requires further DC migration to DC %d", v.DCID)
	default:
		return nil, fmt.Errorf("unexpected login token type %T after migration", result)
	}
}
