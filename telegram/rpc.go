package telegram

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"sync/atomic"
	"time"

	"github.com/mtgo-labs/mtgo/internal/session"
	"github.com/mtgo-labs/mtgo/tg"
	"github.com/mtgo-labs/mtgo/tgerr"
)

type authAttemptContextKey struct{}

// authAttemptGeneration follows transparent RPC retries/migrations and records
// the permanent-key generation that produced the final successful result.
type authAttemptGeneration struct {
	generation atomic.Uint64
}

func (c *Client) withAuthAttempt(ctx context.Context) (context.Context, *authAttemptGeneration) {
	attempt := &authAttemptGeneration{}
	attempt.generation.Store(c.authGeneration.Load())
	return context.WithValue(ctx, authAttemptContextKey{}, attempt), attempt
}

func recordAuthAttemptGeneration(ctx context.Context, generation uint64) {
	if ctx == nil {
		return
	}
	if attempt, ok := ctx.Value(authAttemptContextKey{}).(*authAttemptGeneration); ok && attempt != nil {
		attempt.generation.Store(generation)
	}
}

type clientInvoker struct {
	client *Client
}

func (ci *clientInvoker) RPCInvoke(ctx context.Context, input tg.TLObject, decode func(*tg.Reader) (tg.TLObject, error)) (tg.TLObject, error) {
	cfg := ci.client.config()
	deadline, ok := ctx.Deadline()
	timeout := time.Duration(0)
	if ok {
		timeout = max(time.Until(deadline), 0)
	} else {
		timeout = cfg.ReqTimeout
		if timeout <= 0 {
			timeout = 60 * time.Second
		}
	}
	if timeout < time.Second {
		timeout = time.Second
	}

	retries := max(cfg.Retries, 1)
	query, initializesAPI := prepareAPIQuery(cfg, ci.client.apiInit.Load(), input)

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
	if initializesAPI {
		ci.client.apiInit.Store(true)
	}
	return result, nil
}

func (ci *clientInvoker) RPCInvokeRaw(ctx context.Context, input tg.TLObject) ([]byte, error) {
	cfg := ci.client.config()
	query, initializesAPI := prepareAPIQuery(cfg, ci.client.apiInit.Load(), input)

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
				return ci.client.handleRawMigrationError(ctx, rpcErr, input)
			}
			return nil, err
		}
	}
	if initializesAPI {
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

func prepareAPIQuery(cfg Config, initialized bool, input tg.TLObject) (tg.TLObject, bool) {
	if initialized || !needsInitConnection(input) {
		return input, false
	}
	return wrapInitConnection(cfg, input), true
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
	if _, ok := errors.AsType[*session.DeliveryError](err); ok {
		return true
	}
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
	for {
		if err := c.authLossError(); err != nil {
			return err
		}
		if c.explicitLogout.Load() {
			return ErrNotConnected
		}
		if c.state.IsConnected() {
			return c.waitMainReadiness(ctx)
		}
		if c.state.IsClosed() {
			return ErrClientClosed
		}

		c.mu.RLock()
		ch := c.connChanged
		c.mu.RUnlock()

		// A transition can close and replace connChanged between the first
		// state check and this snapshot. Recheck before blocking so that a
		// connected or terminal client never waits on the replacement channel.
		if err := c.authLossError(); err != nil {
			return err
		}
		if c.explicitLogout.Load() {
			return ErrNotConnected
		}
		if c.state.IsConnected() {
			return c.waitMainReadiness(ctx)
		}
		if c.state.IsClosed() {
			return ErrClientClosed
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ch:
			// Re-evaluate the terminal/connected state and subscribe to the
			// current generation if the signal represented another transition.
		}
	}
}

// retrySessionErr retries fn when it returns a session-death error, waiting for
// reconnection between attempts. When RetryRPCOnReconnect is disabled, fn is
// called exactly once.
func (c *Client) retrySessionErr(ctx context.Context, fn func(*session.Session) error, query ...tg.TLObject) error {
	runAttempt := func(attempt int) (authLoss *authLossState, firstLoss bool, err error) {
		started := time.Now()
		var observedErr error
		var sourceSession *session.Session
		var sourceAuthGeneration uint64
		var pfsRejected bool
		func() {
			c.authDecisionMu.RLock()
			defer c.authDecisionMu.RUnlock()

			c.mu.RLock()
			sess := c.session
			c.mu.RUnlock()
			sourceSession = sess
			sourceAuthGeneration = c.authGeneration.Load()
			observedErr = fn(sess)
			err = observedErr
			if err == nil {
				recordAuthAttemptGeneration(ctx, sourceAuthGeneration)
			}
			if c.activePFSAuthRejection(sess, err) {
				pfsRejected = true
				return
			}
			if err != nil && c.shouldInvalidateMainAuthFrom(sess, err) {
				var accepted bool
				authLoss, firstLoss, accepted = c.latchMainAuthLossFrom(sess, sourceAuthGeneration, err)
				if accepted {
					err = c.authLossResult(authLoss)
				}
			}
		}()

		if pfsRejected {
			err = c.rejectActivePFSKey(sourceSession, err)
		}
		// Give terminal cleanup an independent completion path before invoking
		// application telemetry. A callback may itself call Connect or Start and
		// wait for this loss to finish.
		if authLoss != nil {
			err = c.completeMainAuthInvalidation(authLoss, firstLoss)
		}
		// Telemetry is application code and may call Close or Disconnect. It must
		// run after authDecisionMu is released.
		c.observeRPC(ctx, firstQuery(query), attempt, started, observedErr)
		return authLoss, firstLoss, err
	}

	retryOnReconnect := c.retryRPCOnReconnect(ctx)
	if !retryOnReconnect {
		_, _, err := runAttempt(1)
		return publicDeliveryError(err, firstQuery(query))
	}

	maxRetries := c.config().MaxRPCReconnectRetries
	if maxRetries == 0 {
		maxRetries = 3
	}

	var lastErr error
	for attempt := 0; maxRetries < 0 || attempt <= maxRetries; attempt++ {
		authLoss, _, err := runAttempt(attempt + 1)
		if err == nil {
			return nil
		}
		if authLoss != nil {
			return err
		}
		if errors.Is(err, errTemporaryAuthKeyRejected) {
			if !c.config().ReconnectEnabled {
				return err
			}
			lastErr = err
			if waitErr := c.waitForConnect(ctx); waitErr != nil {
				return fmt.Errorf("%w (last session error: %v)", waitErr, lastErr)
			}
			continue
		}
		if !isSessionDeadErr(err) {
			return err
		}
		// RPCReplaySafe is application code and may call Close or Disconnect.
		// Evaluate it only after authDecisionMu is released.
		if _, ok := errors.AsType[*session.DeliveryError](err); ok && !c.replaySafe(firstQuery(query)) {
			return publicDeliveryError(err, firstQuery(query))
		}
		lastErr = err
		if c.Log != nil {
			c.Log.Debugf("RPC reconnect retry attempt=%d err=%v", attempt+1, err)
		}
		if waitErr := c.waitForConnect(ctx); waitErr != nil {
			return fmt.Errorf("%w (last session error: %v)", waitErr, lastErr)
		}
	}
	return fmt.Errorf("session closed after %d reconnect retries: %w", maxRetries, lastErr)
}

func publicDeliveryError(err error, query tg.TLObject) error {
	deliveryErr, ok := errors.AsType[*session.DeliveryError](err)
	if !ok {
		return err
	}
	state := RPCDeliveryUnknown
	if deliveryErr.State == session.DeliveryReceived {
		state = RPCDeliveryReceived
	}
	return &RPCDeliveryError{
		Method: rpcQueryName(query),
		State:  state,
		Err:    err,
	}
}

func firstQuery(queries []tg.TLObject) tg.TLObject {
	if len(queries) == 0 {
		return nil
	}
	return queries[0]
}

func (c *Client) replaySafe(query tg.TLObject) bool {
	query = unwrapRPCQuery(query)
	if query == nil {
		return false
	}
	switch query.(type) {
	case *tg.UploadSaveFilePartRequest, *tg.UploadSaveBigFilePartRequest:
		// A file part is identified by file_id and part index. Replaying the
		// exact request replaces the same part instead of duplicating an
		// application-level mutation.
		return true
	}
	if callback := c.config().RPCReplaySafe; callback != nil && callback(query) {
		return true
	}
	name := rpcQueryName(query)
	for _, marker := range []string{
		"AccountGet", "AuthExportAuthorization", "BotsGet", "ChannelsGet",
		"ChatlistsGet", "ContactsGet", "ContactsResolve", "ContactsSearch",
		"HelpGet", "LangpackGet", "MessagesCheck", "MessagesGet", "MessagesSearch",
		"PaymentsGet", "PhoneGet", "PhotosGet", "PremiumGet", "StatsGet",
		"StatsLoad", "StickersGet", "StoriesGet", "StoriesSearch", "UpdatesGet",
		"UploadGet", "UsersGet",
	} {
		if strings.HasPrefix(name, marker) {
			return true
		}
	}
	return false
}

func unwrapRPCQuery(query tg.TLObject) tg.TLObject {
	for query != nil {
		switch wrapped := query.(type) {
		case *tg.InvokeWithLayerRequest:
			query = wrapped.Query
		case *tg.InitConnectionRequest:
			query = wrapped.Query
		case *tg.InvokeAfterMsgRequest:
			query = wrapped.Query
		case *tg.InvokeAfterMsgsRequest:
			query = wrapped.Query
		case *tg.InvokeWithoutUpdatesRequest:
			query = wrapped.Query
		case *tg.InvokeWithTakeoutRequest:
			query = wrapped.Query
		case *tg.InvokeWithBusinessConnectionRequest:
			query = wrapped.Query
		default:
			return query
		}
	}
	return nil
}

func rpcQueryName(query tg.TLObject) string {
	query = unwrapRPCQuery(query)
	if query == nil {
		return "unknown"
	}
	t := reflect.TypeOf(query)
	if t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	return strings.TrimSuffix(t.Name(), "Request")
}
