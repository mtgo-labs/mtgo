package tg

import (
	"context"
)

type Invoker interface {
	RPCInvoke(ctx context.Context, input TLObject, decode func(*Reader) (TLObject, error)) (TLObject, error)
	RPCInvokeRaw(ctx context.Context, input TLObject) ([]byte, error)
}

type InvokerFunc func(ctx context.Context, input TLObject, decode func(*Reader) (TLObject, error)) (TLObject, error)

func (f InvokerFunc) RPCInvoke(ctx context.Context, input TLObject, decode func(*Reader) (TLObject, error)) (TLObject, error) {
	return f(ctx, input, decode)
}

func (f InvokerFunc) RPCInvokeRaw(ctx context.Context, input TLObject) ([]byte, error) {
	return nil, nil
}

// Client wraps an Invoker and provides a high-level RPC interface.
type Client struct {
	rpc Invoker
}

// RPC returns the underlying Invoker used by the client.
func (c *RPCClient) RPC() Invoker { return c.rpc }

// NewClient creates a new Client backed by the given Invoker.
func NewClient(invoker Invoker) *Client {
	return &Client{rpc: invoker}
}

// Invoke performs an RPC call by delegating to the underlying Invoker.
func (c *RPCClient) Invoke(ctx context.Context, input TLObject, decode func(*Reader) (TLObject, error)) (TLObject, error) {
	return c.rpc.RPCInvoke(ctx, input, decode)
}

// InvokeWithRawResult sends a TLObject query and returns the raw MTProto
// rpc_result result:Object payload bytes without gzip unpacking or TL decoding.
func (c *RPCClient) InvokeWithRawResult(ctx context.Context, input TLObject) ([]byte, error) {
	return c.rpc.RPCInvokeRaw(ctx, input)
}

// InvokeWithBytes is deprecated. Use [RPCClient.InvokeWithRawResult].
func (c *RPCClient) InvokeWithBytes(ctx context.Context, input TLObject) ([]byte, error) {
	return c.InvokeWithRawResult(ctx, input)
}
