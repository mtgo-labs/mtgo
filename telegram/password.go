package telegram

import (
	"context"
	"errors"
	"fmt"
	"math/big"

	"github.com/mtgo-labs/mtgo/internal/crypto"
	"github.com/mtgo-labs/mtgo/tg"
)

// ErrPasswordAlreadyEnabled is returned when attempting to enable a cloud password
// but one is already set on the account.
var ErrPasswordAlreadyEnabled = errors.New("cloud password already enabled")

// ErrPasswordNotEnabled is returned when attempting to change or remove a cloud password
// but no cloud password is currently set on the account.
var ErrPasswordNotEnabled = errors.New("cloud password not enabled")

func computeNewPasswordHash(algo *tg.PasswordKdfAlgoSha256sha256pbkdf2hmacsha512iter100000sha256modPow, password string) ([]byte, error) {
	x := new(big.Int).SetBytes(crypto.ComputePasswordHash(password, algo.Salt1, algo.Salt2))
	p := new(big.Int).SetBytes(algo.P)
	g := big.NewInt(int64(algo.G))
	hash := new(big.Int).Exp(g, x, p).Bytes()
	if len(hash) < 256 {
		padded := make([]byte, 256)
		copy(padded[256-len(hash):], hash)
		hash = padded
	}
	if len(hash) > 256 {
		hash = hash[len(hash)-256:]
	}
	return hash, nil
}

// EnableCloudPassword enables two-factor authentication by setting a cloud password
// on the account. password is the new password to set, and hint is an optional
// recovery hint. Returns ErrPasswordAlreadyEnabled if a password is already set,
// or an error if the client is not connected or the RPC call fails.
//
// Example:
//
//	ctx := context.Background()
//	err := client.EnableCloudPassword(ctx, "my-s3cret-p4ss", "favorite color")
//	if err != nil {
//	    log.Fatal(err)
//	}
func (c *Client) EnableCloudPassword(ctx context.Context, password, hint string) error {
	if err := c.ensureConnected(); err != nil {
		return err
	}

	c.Log.Debug("enabling cloud password")
	rpc := c.Raw()
	pwd, err := rpc.AccountGetPassword(ctx)
	if err != nil {
		return fmt.Errorf("get password: %w", err)
	}

	if pwd.HasPassword {
		c.Log.Warn("password: cloud password already enabled")
		return ErrPasswordAlreadyEnabled
	}

	algo, ok := pwd.NewAlgo.(*tg.PasswordKdfAlgoSha256sha256pbkdf2hmacsha512iter100000sha256modPow)
	if !ok {
		return fmt.Errorf("unsupported password KDF algorithm: %T", pwd.NewAlgo)
	}

	newHash, err := computeNewPasswordHash(algo, password)
	if err != nil {
		return fmt.Errorf("compute new password hash: %w", err)
	}

	_, err = rpc.AccountUpdatePasswordSettings(ctx, &tg.AccountUpdatePasswordSettingsRequest{
		Password: &tg.InputCheckPasswordEmpty{},
		NewSettings: &tg.AccountPasswordInputSettings{
			NewAlgo:         algo,
			NewPasswordHash: newHash,
			Hint:            hint,
		},
	})
	if err != nil {
		c.Log.Warnf("password: enable cloud password failed err=%v", err)
		return err
	}
	c.Log.Info("password: cloud password enabled")
	return err
}

// ChangeCloudPassword changes the existing cloud password to newPassword with the
// given newHint. currentPassword must match the currently set password.
// Returns ErrPasswordNotEnabled if no cloud password is set, or an error if the
// client is not connected, the current password is incorrect, or the RPC call fails.
//
// Example:
//
//	ctx := context.Background()
//	err := client.ChangeCloudPassword(ctx, "old-password", "new-password", "new hint")
//	if err != nil {
//	    log.Fatal(err)
//	}
func (c *Client) ChangeCloudPassword(ctx context.Context, currentPassword, newPassword, newHint string) error {
	if err := c.ensureConnected(); err != nil {
		return err
	}

	c.Log.Debug("changing cloud password")
	rpc := c.Raw()
	pwd, err := rpc.AccountGetPassword(ctx)
	if err != nil {
		return fmt.Errorf("get password: %w", err)
	}

	if !pwd.HasPassword {
		return ErrPasswordNotEnabled
	}

	srpInput, err := c.computeSRP(ctx, currentPassword)
	if err != nil {
		return fmt.Errorf("compute SRP: %w", err)
	}

	algo, ok := pwd.NewAlgo.(*tg.PasswordKdfAlgoSha256sha256pbkdf2hmacsha512iter100000sha256modPow)
	if !ok {
		return fmt.Errorf("unsupported password KDF algorithm: %T", pwd.NewAlgo)
	}

	newHash, err := computeNewPasswordHash(algo, newPassword)
	if err != nil {
		return fmt.Errorf("compute new password hash: %w", err)
	}

	_, err = rpc.AccountUpdatePasswordSettings(ctx, &tg.AccountUpdatePasswordSettingsRequest{
		Password: srpInput,
		NewSettings: &tg.AccountPasswordInputSettings{
			NewAlgo:         algo,
			NewPasswordHash: newHash,
			Hint:            newHint,
		},
	})
	if err != nil {
		c.Log.Warnf("password: change cloud password failed err=%v", err)
		return err
	}
	c.Log.Info("password: cloud password changed")
	return err
}

// RemoveCloudPassword disables two-factor authentication by removing the cloud password
// from the account. password must match the currently set password.
// Returns ErrPasswordNotEnabled if no cloud password is set, or an error if the
// client is not connected, the password is incorrect, or the RPC call fails.
//
// Example:
//
//	ctx := context.Background()
//	err := client.RemoveCloudPassword(ctx, "my-current-password")
//	if err != nil {
//	    log.Fatal(err)
//	}
func (c *Client) RemoveCloudPassword(ctx context.Context, password string) error {
	if err := c.ensureConnected(); err != nil {
		return err
	}

	c.Log.Debug("removing cloud password")
	rpc := c.Raw()
	pwd, err := rpc.AccountGetPassword(ctx)
	if err != nil {
		return fmt.Errorf("get password: %w", err)
	}

	if !pwd.HasPassword {
		return ErrPasswordNotEnabled
	}

	srpInput, err := c.computeSRP(ctx, password)
	if err != nil {
		return fmt.Errorf("compute SRP: %w", err)
	}

	_, err = rpc.AccountUpdatePasswordSettings(ctx, &tg.AccountUpdatePasswordSettingsRequest{
		Password:    srpInput,
		NewSettings: &tg.AccountPasswordInputSettings{},
	})
	if err != nil {
		c.Log.Warnf("password: remove cloud password failed err=%v", err)
		return err
	}
	c.Log.Info("password: cloud password removed")
	return err
}
