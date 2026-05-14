package tg

import (
	"context"
	"io"
)

// Invoker is the interface for types that can perform TL RPC calls.
type Invoker interface {
	RPCInvoke(ctx context.Context, input TLObject, decode func(io.Reader) (TLObject, error)) (TLObject, error)
}

// InvokerFunc implements Invoker as a function, allowing ordinary functions
// to be used where an Invoker is expected.
type InvokerFunc func(ctx context.Context, input TLObject, decode func(io.Reader) (TLObject, error)) (TLObject, error)

// RPCInvoke implements Invoker by calling the function itself.
func (f InvokerFunc) RPCInvoke(ctx context.Context, input TLObject, decode func(io.Reader) (TLObject, error)) (TLObject, error) {
	return f(ctx, input, decode)
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
func (c *RPCClient) Invoke(ctx context.Context, input TLObject, decode func(io.Reader) (TLObject, error)) (TLObject, error) {
	return c.rpc.RPCInvoke(ctx, input, decode)
}
