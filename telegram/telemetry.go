package telegram

import (
	"context"
	"errors"
	"time"

	"github.com/mtgo-labs/mtgo/internal/session"
	"github.com/mtgo-labs/mtgo/tg"
	"github.com/mtgo-labs/mtgo/tgerr"
)

// Telemetry receives structured, secret-free MTProto lifecycle observations.
// Implementations must be concurrency-safe and should not block the caller.
type Telemetry interface {
	ObserveRPC(context.Context, RPCObservation)
	ObserveConnection(context.Context, ConnectionObservation)
}

// RPCObservation describes one physical RPC attempt. Query payloads and auth
// material are deliberately excluded.
type RPCObservation struct {
	Method        string
	DCID          int
	Attempt       int
	StartedAt     time.Time
	EndedAt       time.Time
	Error         error
	ErrorClass    string
	DeliveryState RPCDeliveryState
	FloodWait     time.Duration
}

// ConnectionObservation describes a transport or connection lifecycle event.
type ConnectionObservation struct {
	Kind      string
	Endpoint  string
	StartedAt time.Time
	EndedAt   time.Time
	Error     error
}

func (c *Client) observeRPC(ctx context.Context, query tg.TLObject, attempt int, started time.Time, err error) {
	observer := c.config().Telemetry
	if observer == nil {
		return
	}
	observation := RPCObservation{
		Method:     rpcQueryName(query),
		DCID:       c.homeDC(),
		Attempt:    attempt,
		StartedAt:  started,
		EndedAt:    time.Now(),
		Error:      err,
		ErrorClass: telemetryErrorClass(err),
	}
	if deliveryErr, ok := errors.AsType[*session.DeliveryError](err); ok {
		observation.DeliveryState = RPCDeliveryUnknown
		if deliveryErr.State == session.DeliveryReceived {
			observation.DeliveryState = RPCDeliveryReceived
		}
	}
	if rpcErr, ok := tgerr.As(err); ok && rpcErr.Type == "FLOOD_WAIT" {
		observation.FloodWait = time.Duration(rpcErr.Argument) * time.Second
	}
	observer.ObserveRPC(ctx, observation)
}

func telemetryErrorClass(err error) string {
	if err == nil {
		return "ok"
	}
	if rpcErr, ok := tgerr.As(err); ok {
		if rpcErr.Code == 303 {
			return "migrate"
		}
		if rpcErr.Code == 420 {
			return "rate_limited"
		}
		if rpcErr.Code >= 500 {
			return "server"
		}
		return "rpc"
	}
	return session.ClassifyError(err).String()
}
