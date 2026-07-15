package telegram

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/mtgo-labs/mtgo/internal/session"
	"github.com/mtgo-labs/mtgo/tg"
	"github.com/mtgo-labs/mtgo/tgerr"
)

type clientInvoker struct {
	client *Client
}

func (ci *clientInvoker) RPCInvoke(ctx context.Context, input tg.TLObject, decode func(*tg.Reader) (tg.TLObject, error)) (tg.TLObject, error) {
	cfg := ci.client.config()
	deadline, ok := ctx.Deadline()
	timeout := time.Duration(0)
	if ok {
		timeout = time.Until(deadline)
		if timeout < 0 {
			timeout = 0
		}
	} else {
		timeout = cfg.ReqTimeout
		if timeout <= 0 {
			timeout = 60 * time.Second
		}
	}
	if timeout < time.Second {
		timeout = time.Second
	}

	retries := cfg.Retries
	if retries < 1 {
		retries = 1
	}
	apiInit := ci.client.apiInit.Load()

	query := input
	if !apiInit && needsInitConnection(input) {
		query = wrapInitConnection(cfg, input)
	}

	ci.client.Log.Debugf("RPC invoke method=%T timeout=%s", input, timeout)

	result, err := ci.client.Invoke(ctx, query, retries, timeout)
	if err != nil {
		return nil, err
	}
	if rpcErr, ok := result.(*tg.RPCError); ok {
		ci.client.Log.Warnf("RPC error code=%d msg=%s", rpcErr.ErrorCode, rpcErr.ErrorMessage)
		parsed := tgerr.New(int(rpcErr.ErrorCode), rpcErr.ErrorMessage)
		if parsed.Code == 303 {
			if shouldReturnMigrationToCaller(input, parsed) {
				return nil, parsed
			}
			return ci.client.handleMigrationError(ctx, parsed, input)
		}
		// Auto-retry FLOOD_WAIT below SleepThreshold.
		if wait, fwOk := tgerr.AsFloodWait(parsed); fwOk && cfg.SleepThreshold > 0 && wait <= cfg.SleepThreshold {
			ci.client.Log.Debugf("RPC flood-wait %s: auto-retry within threshold %s", wait, cfg.SleepThreshold)
			select {
			case <-time.After(wait):
			case <-ctx.Done():
				return nil, ctx.Err()
			}
			result, err = ci.client.Invoke(ctx, query, retries, timeout)
			if err != nil {
				return nil, err
			}
			if rpcErr2, ok := result.(*tg.RPCError); ok {
				return nil, tgerr.New(int(rpcErr2.ErrorCode), rpcErr2.ErrorMessage)
			}
		} else {
			return nil, parsed
		}
	}
	if result == nil {
		ci.client.Log.Warnf("RPC nil result method=%T", input)
		return nil, fmt.Errorf("telegram: nil RPC result for %T", input)
	}
	if !apiInit && needsInitConnection(input) {
		ci.client.apiInit.Store(true)
	}
	return result, nil
}

func (ci *clientInvoker) RPCInvokeRaw(ctx context.Context, input tg.TLObject) ([]byte, error) {
	apiInit := ci.client.apiInit.Load()
	cfg := ci.client.config()

	query := input
	if !apiInit && needsInitConnection(input) {
		query = wrapInitConnection(cfg, input)
	}

	data, err := ci.client.InvokeWithRawResult(ctx, query)
	if err != nil {
		// Auto-retry FLOOD_WAIT below SleepThreshold.
		if wait, fwOk := tgerr.AsFloodWait(err); fwOk && cfg.SleepThreshold > 0 && wait <= cfg.SleepThreshold {
			ci.client.Log.Debugf("RPC flood-wait %s: auto-retry within threshold %s", wait, cfg.SleepThreshold)
			select {
			case <-time.After(wait):
			case <-ctx.Done():
				return nil, ctx.Err()
			}
			data, err = ci.client.InvokeWithRawResult(ctx, query)
		}
		if err != nil {
			if rpcErr, ok := tgerr.As(err); ok && rpcErr.Code == 303 {
				if shouldReturnMigrationToCaller(input, rpcErr) {
					return nil, rpcErr
				}
				_, migErr := ci.client.handleMigrationError(ctx, rpcErr, input)
				return nil, migErr
			}
			return nil, err
		}
	}
	if !apiInit && needsInitConnection(input) {
		ci.client.apiInit.Store(true)
	}
	return data, nil
}

func shouldReturnMigrationToCaller(input tg.TLObject, err *tgerr.Error) bool {
	if input == nil || err == nil {
		return false
	}
	return err.Code == 303 &&
		err.Type == "USER_MIGRATE" &&
		input.ConstructorID() == tg.AuthImportBotAuthorizationTypeID
}

func needsInitConnection(input tg.TLObject) bool {
	if input == nil {
		return false
	}
	switch input.ConstructorID() {
	case tg.InvokeWithLayerTypeID, tg.InitConnectionTypeID:
		return false
	default:
		return true
	}
}

func wrapInitConnection(cfg Config, input tg.TLObject) tg.TLObject {
	return &tg.InvokeWithLayerRequest{
		Layer: tg.Layer,
		Query: &tg.InitConnectionRequest{
			APIID:          cfg.APIID,
			DeviceModel:    cfg.Device.DeviceModel,
			SystemVersion:  cfg.Device.SystemVersion,
			AppVersion:     cfg.Device.AppVersion,
			SystemLangCode: cfg.Device.SystemLangCode,
			LangPack:       cfg.Device.LangPack,
			LangCode:       cfg.Device.LangCode,
			Query:          input,
		},
	}
}

// Raw returns a low-level tg.RPCClient that sends TL objects directly to the
// Telegram server. Use this when the higher-level RPC method is insufficient and
// you need to invoke arbitrary TL constructors or interact with the raw MTProto
// layer.
//
// The returned client transparently wraps every request in an initConnection
// envelope on the first call after connecting, so callers do not need to handle
// connection initialisation themselves.
//
// Example:
//
//	rpc := client.Raw()
//	peers, err := rpc.ContactsResolveUsername(ctx, &tg.ContactsResolveUsernameRequest{
//	    Username: "durov",
//	})
func (c *Client) Raw() *tg.RPCClient {
	if inv := c.testInvoker; inv != nil {
		return tg.NewRPCClient(inv)
	}

	c.mu.Lock()
	if c.invokerCache != nil {
		c.mu.Unlock()
		return c.invokerCache
	}
	mws := c.invokerMiddlewares

	base := tg.Invoker(&clientInvoker{client: c})
	for i := len(mws) - 1; i >= 0; i-- {
		base = mws[i](base)
	}
	rpcClient := tg.NewRPCClient(base)
	c.invokerCache = rpcClient
	c.mu.Unlock()

	return rpcClient
}

// RPC is an alias for [Client.Raw]. It returns the low-level RPC client used to
// invoke Telegram TL methods. Prefer this method name when the intent is to make
// typed API calls via the generated tg.RPCClient surface.
//
// Example:
//
//	result, err := client.RPC().MessagesGetMessages(ctx, &tg.MessagesGetMessagesRequest{
//	    ID: []tg.InputMessageClass{&tg.InputMessageID{ID: 42}},
//	})
func (c *Client) RPC() *tg.RPCClient { return c.Raw() }

// InvokeJSON sends a raw Telegram API call encoded as JSON. functionName is the
// exact TL method name (e.g. "messages.SendMessage"). payload is the JSON-encoded
// request body. When useSnakeCase is true, field names are expected in snake_case
// (matching the TL schema); otherwise they are expected in camelCase. The response
// is returned as a JSON byte slice.
//
// This is useful for callers that construct requests dynamically or prefer JSON
// over the generated Go types.
//
// Returns the raw JSON response body from the server, or an error if the RPC call
// fails or the server returns an RPC error.
//
// Example:
//
//	ctx := context.Background()
//	payload := []byte(`{"peer":"inputPeerSelf","message":"hello","random_id":123}`)
//	resp, err := client.InvokeJSON(ctx, "messages.SendMessage", payload, false)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	fmt.Println(string(resp))
func (c *Client) InvokeJSON(ctx context.Context, functionName string, payload []byte, useSnakeCase bool) ([]byte, error) {
	jc := NewJSONClient(c.Raw())
	return jc.InvokeJSON(ctx, functionName, payload, useSnakeCase)
}

// isSessionDeadErr reports whether err indicates the underlying session is no
// longer usable (closed, draining, or disconnected). These are the errors that
// trigger reconnect-retry when RetryRPCOnReconnect is enabled.
func isSessionDeadErr(err error) bool {
	return errors.Is(err, session.ErrSessionClosed) ||
		errors.Is(err, session.ErrDraining) ||
		errors.Is(err, session.ErrNotConnected) ||
		errors.Is(err, ErrNotConnected)
}

// waitForConnect blocks until the client reports a connected state, the context
// is cancelled, or the client is closed. Returns nil when connected.
// Uses a channel-based signal (connChanged) to wake immediately on reconnect
// instead of polling, matching gotd/td's pattern.
func (c *Client) waitForConnect(ctx context.Context) error {
	if c.state.IsConnected() {
		return nil
	}
	if c.state.IsClosed() {
		return ErrClientClosed
	}

	c.mu.RLock()
	ch := c.connChanged
	c.mu.RUnlock()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-ch:
		// connChanged was closed — connection transition happened.
		// Check whether we became connected or closed.
		switch {
		case c.state.IsConnected():
			return nil
		case c.state.IsClosed():
			return ErrClientClosed
		default:
			// State is neither connected nor closed after signal.
			// The reconnect may have failed; the caller will see the
			// error from the session and retry via retrySessionErr.
			return ErrNotConnected
		}
	}
}

// retrySessionErr retries fn when it returns a session-death error, waiting for
// reconnection between attempts. When RetryRPCOnReconnect is disabled, fn is
// called exactly once.
func (c *Client) retrySessionErr(ctx context.Context, fn func() error) error {
	if !c.config().RetryRPCOnReconnect {
		return fn()
	}

	maxRetries := c.config().MaxRPCReconnectRetries
	if maxRetries == 0 {
		maxRetries = 3
	}

	var lastErr error
	for attempt := 0; maxRetries < 0 || attempt <= maxRetries; attempt++ {
		err := fn()
		if err == nil {
			return nil
		}
		if !isSessionDeadErr(err) {
			return err
		}
		lastErr = err
		if c.Log != nil {
			c.Log.Debugf("RPC reconnect retry attempt=%d err=%v", attempt+1, err)
		}
		if waitErr := c.waitForConnect(ctx); waitErr != nil {
			if lastErr != nil {
				return fmt.Errorf("%w (last session error: %v)", waitErr, lastErr)
			}
			return waitErr
		}
	}
	return fmt.Errorf("session closed after %d reconnect retries: %w", maxRetries, lastErr)
}
