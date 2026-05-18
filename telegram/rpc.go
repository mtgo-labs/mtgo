package telegram

import (
	"context"
	"fmt"
	"time"

	"github.com/mtgo-labs/mtgo/tg"
	"github.com/mtgo-labs/mtgo/tgerr"
)

type clientInvoker struct {
	client *Client
}

func (ci *clientInvoker) RPCInvoke(ctx context.Context, input tg.TLObject, decode func(*tg.Reader) (tg.TLObject, error)) (tg.TLObject, error) {
	deadline, ok := ctx.Deadline()
	timeout := time.Duration(0)
	if ok {
		timeout = time.Until(deadline)
		if timeout < 0 {
			timeout = 0
		}
	} else {
		timeout = ci.client.cfg.ReqTimeout
		if timeout <= 0 {
			timeout = 60 * time.Second
		}
	}
	if timeout < time.Second {
		timeout = time.Second
	}

	retries := ci.client.cfg.Retries
	if retries < 1 {
		retries = 1
	}
	ci.client.mu.RLock()
	apiInit := ci.client.apiInit
	cfg := ci.client.cfg
	ci.client.mu.RUnlock()

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
			return ci.client.handleMigrationError(parsed, input)
		}
		return nil, parsed
	}
	if result == nil {
		ci.client.Log.Warnf("RPC nil result method=%T", input)
		return nil, fmt.Errorf("telegram: nil RPC result for %T", input)
	}
	if !apiInit && needsInitConnection(input) {
		ci.client.mu.Lock()
		ci.client.apiInit = true
		ci.client.mu.Unlock()
	}
	return result, nil
}

func (ci *clientInvoker) RPCInvokeRaw(ctx context.Context, input tg.TLObject) ([]byte, error) {
	ci.client.mu.RLock()
	apiInit := ci.client.apiInit
	cfg := ci.client.cfg
	ci.client.mu.RUnlock()

	query := input
	if !apiInit && needsInitConnection(input) {
		query = wrapInitConnection(cfg, input)
	}

	data, err := ci.client.InvokeWithRawByte(ctx, query)
	if err != nil {
		return nil, err
	}
	if !apiInit && needsInitConnection(input) {
		ci.client.mu.Lock()
		ci.client.apiInit = true
		ci.client.mu.Unlock()
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

	c.mu.RLock()
	if c.invokerCache != nil {
		c.mu.RUnlock()
		return c.invokerCache
	}
	mws := c.invokerMiddlewares
	c.mu.RUnlock()

	base := tg.Invoker(&clientInvoker{client: c})
	for i := len(mws) - 1; i >= 0; i-- {
		base = mws[i](base)
	}
	rpcClient := tg.NewRPCClient(base)

	c.mu.Lock()
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
