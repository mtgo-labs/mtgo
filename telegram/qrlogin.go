package telegram

import (
	"context"
	"fmt"

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
	// User is set when checking the token completed authorization, including
	// when Telegram redirected the login to the account's home DC. Token is
	// empty in that case.
	User *types.User
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
	if err := c.ensureConnectedContext(ctx); err != nil {
		return nil, err
	}
	ctx, authAttempt := c.withAuthAttempt(ctx)

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
		return c.qrLoginMigrateToDC(ctx, int(v.DCID), append([]byte(nil), v.Token...))
	case *tg.AuthLoginTokenSuccess:
		return c.qrLoginResult(v, authAttempt.generation.Load())
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
	if err := c.ensureConnectedContext(ctx); err != nil {
		return nil, err
	}
	ctx, authAttempt := c.withAuthAttempt(ctx)

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
		user := types.ParseUser(auth.User)
		if err := c.commitAuthorizedUser(user, authAttempt.generation.Load()); err != nil {
			return nil, err
		}
		c.Log.Info("auth: QR login successful")
		return user, nil
	case *tg.AuthLoginToken:
		return nil, fmt.Errorf("QR code not yet scanned, retry later (expires in %d seconds)", v.Expires)
	case *tg.AuthLoginTokenMigrateTo:
		migrated, err := c.qrLoginMigrateToDC(ctx, int(v.DCID), append([]byte(nil), v.Token...))
		if err != nil {
			return nil, err
		}
		if migrated.User == nil {
			return nil, fmt.Errorf("QR code not yet scanned, retry later (expires in %d seconds)", migrated.Expires)
		}
		return migrated.User, nil
	default:
		return nil, fmt.Errorf("unexpected login token type %T", result)
	}
}

// qrLoginMigrateToDC reconnects with a fresh permanent key on the redirected
// DC and imports exactly the login token supplied by Telegram.
func (c *Client) qrLoginMigrateToDC(ctx context.Context, targetDC int, token []byte) (*QRLoginToken, error) {
	c.Log.Infof("auth: QR login migrating to DC %d", targetDC)

	c.mu.RLock()
	st := c.storage
	c.mu.RUnlock()
	if st == nil {
		return nil, fmt.Errorf("QR login DC migration: no storage available")
	}

	var migrated *QRLoginToken
	err := c.migration.Do(ctx, targetDC, func(ctx context.Context) error {
		if c.homeDC() != targetDC || !c.IsConnected() {
			if err := c.switchPrimaryDC(ctx, targetDC, st, nil); err != nil {
				return fmt.Errorf("reconnect to DC %d: %w", targetDC, err)
			}
		}
		attemptCtx, authAttempt := c.withAuthAttempt(ctx)
		result, err := c.Raw().AuthImportLoginToken(attemptCtx, &tg.AuthImportLoginTokenRequest{Token: token})
		if err != nil {
			return fmt.Errorf("import login token on DC %d: %w", targetDC, err)
		}
		migrated, err = c.qrLoginResult(result, authAttempt.generation.Load())
		return err
	})
	if err != nil {
		return nil, err
	}
	return migrated, nil
}

func (c *Client) qrLoginResult(result tg.LoginTokenClass, authGeneration uint64) (*QRLoginToken, error) {
	switch v := result.(type) {
	case *tg.AuthLoginToken:
		return &QRLoginToken{Token: v.Token, Expires: v.Expires}, nil
	case *tg.AuthLoginTokenSuccess:
		auth, ok := v.Authorization.(*tg.AuthAuthorization)
		if !ok {
			return nil, fmt.Errorf("unexpected authorization type %T", v.Authorization)
		}
		user := types.ParseUser(auth.User)
		if err := c.commitAuthorizedUser(user, authGeneration); err != nil {
			return nil, err
		}
		c.Log.Info("auth: QR login successful")
		return &QRLoginToken{User: user}, nil
	case *tg.AuthLoginTokenMigrateTo:
		return nil, fmt.Errorf("QR login requires further DC migration to DC %d", v.DCID)
	default:
		return nil, fmt.Errorf("unexpected login token type %T after migration", result)
	}
}
