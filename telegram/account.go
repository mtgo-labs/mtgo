package telegram

import (
	"context"
	"fmt"

	"github.com/mtgo-labs/mtgo/tg"
)

// SetPrivacy sets the privacy rules for the given privacy key.
// key specifies which privacy setting to update (e.g. status visibility, phone number),
// and rules is the list of allow/deny rules to apply. Returns an error if the client
// is not connected or the RPC call fails.
//
// Example:
//
//	err := client.SetPrivacy(ctx,
//	    &tg.InputPrivacyKeyStatusTimestamp{},
//	    []tg.InputPrivacyRuleClass{&tg.InputPrivacyValueAllowAll{}},
//	)
//	if err != nil {
//	    log.Fatal(err)
//	}
func (c *Client) SetPrivacy(ctx context.Context, key tg.InputPrivacyKeyClass, rules []tg.InputPrivacyRuleClass) error {
	if err := c.ensureConnected(); err != nil {
		return err
	}

	c.Log.Debug("SetPrivacy")

	rpc := c.Raw()
	_, err := rpc.AccountSetPrivacy(ctx, &tg.AccountSetPrivacyRequest{
		Key:   key,
		Rules: rules,
	})
	if err != nil {
		c.Log.Warnf("SetPrivacy failed err=%v", err)
		return fmt.Errorf("set privacy: %w", err)
	}
	return nil
}

// GetPrivacy returns the current privacy rules for the given privacy key.
// key specifies which privacy setting to query. Returns the list of active
// PrivacyRuleClass entries or an error if the client is not connected or the RPC call fails.
//
// Example:
//
//	rules, err := client.GetPrivacy(ctx, &tg.InputPrivacyKeyStatusTimestamp{})
//	if err != nil {
//	    log.Fatal(err)
//	}
//	for _, r := range rules {
//	    fmt.Printf("Rule: %T\n", r)
//	}
func (c *Client) GetPrivacy(ctx context.Context, key tg.InputPrivacyKeyClass) ([]tg.PrivacyRuleClass, error) {
	if err := c.ensureConnected(); err != nil {
		return nil, err
	}

	c.Log.Debug("GetPrivacy")

	rpc := c.Raw()
	result, err := rpc.AccountGetPrivacy(ctx, &tg.AccountGetPrivacyRequest{
		Key: key,
	})
	if err != nil {
		c.Log.Warnf("GetPrivacy failed err=%v", err)
		return nil, fmt.Errorf("get privacy: %w", err)
	}
	return result.Rules, nil
}

// SetGlobalPrivacySettings applies the given global privacy settings to the account.
// settings must be non-nil. Returns an error if the client is not connected,
// settings is nil, or the RPC call fails.
//
// Example:
//
//	err := client.SetGlobalPrivacySettings(ctx, &tg.GlobalPrivacySettings{
//	    ArchiveAndMuteNewNoncontactPeers: true,
//	})
//	if err != nil {
//	    log.Fatal(err)
//	}
func (c *Client) SetGlobalPrivacySettings(ctx context.Context, settings *tg.GlobalPrivacySettings) error {
	if err := c.ensureConnected(); err != nil {
		return err
	}

	c.Log.Debug("SetGlobalPrivacySettings")

	if settings == nil {
		return ErrPrivacySettingsRequired
	}

	rpc := c.Raw()
	_, err := rpc.AccountSetGlobalPrivacySettings(ctx, &tg.AccountSetGlobalPrivacySettingsRequest{
		Settings: settings,
	})
	if err != nil {
		c.Log.Warnf("SetGlobalPrivacySettings failed err=%v", err)
		return fmt.Errorf("set global privacy settings: %w", err)
	}
	return nil
}

// GetGlobalPrivacySettings returns the current global privacy settings for the account.
// Returns the GlobalPrivacySettings or an error if the client is not connected
// or the RPC call fails.
//
// Example:
//
//	settings, err := client.GetGlobalPrivacySettings(ctx)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	fmt.Printf("Archive and mute: %v\n", settings.ArchiveAndMuteNewNoncontactPeers)
func (c *Client) GetGlobalPrivacySettings(ctx context.Context) (*tg.GlobalPrivacySettings, error) {
	if err := c.ensureConnected(); err != nil {
		return nil, err
	}

	c.Log.Debug("GetGlobalPrivacySettings")

	rpc := c.Raw()
	result, err := rpc.AccountGetGlobalPrivacySettings(ctx)
	if err != nil {
		c.Log.Warnf("GetGlobalPrivacySettings failed err=%v", err)
		return nil, fmt.Errorf("get global privacy settings: %w", err)
	}
	return result, nil
}

// SetAccountTTL sets the number of days after which the account will self-destruct
// if the user does not log in. Returns an error if the client is not connected
// or the RPC call fails.
//
// Example:
//
//	err := client.SetAccountTTL(ctx, 365)
//	if err != nil {
//	    log.Fatal(err)
//	}
func (c *Client) SetAccountTTL(ctx context.Context, days int32) error {
	if err := c.ensureConnected(); err != nil {
		return err
	}

	c.Log.Debugf("SetAccountTTL days=%d", days)

	rpc := c.Raw()
	_, err := rpc.AccountSetAccountTTL(ctx, &tg.AccountSetAccountTTLRequest{
		TTL: &tg.AccountDaysTTL{Days: days},
	})
	if err != nil {
		c.Log.Warnf("SetAccountTTL failed err=%v", err)
		return fmt.Errorf("set account ttl: %w", err)
	}
	return nil
}

// GetAccountTTL returns the number of days after which the account will self-destruct
// if the user does not log in. Returns the TTL in days or an error if the client
// is not connected or the RPC call fails.
//
// Example:
//
//	days, err := client.GetAccountTTL(ctx)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	fmt.Printf("Account TTL: %d days\n", days)
func (c *Client) GetAccountTTL(ctx context.Context) (int32, error) {
	if err := c.ensureConnected(); err != nil {
		return 0, err
	}

	c.Log.Debug("GetAccountTTL")

	rpc := c.Raw()
	result, err := rpc.AccountGetAccountTTL(ctx)
	if err != nil {
		c.Log.Warnf("GetAccountTTL failed err=%v", err)
		return 0, fmt.Errorf("get account ttl: %w", err)
	}
	return result.Days, nil
}
