// Package otel adapts MTGO's dependency-free telemetry interface to
// OpenTelemetry tracing and metrics.
package otel

import (
	"context"
	"errors"

	mtgo "github.com/mtgo-labs/mtgo/telegram"
	gootel "go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
)

const instrumentationName = "github.com/mtgo-labs/mtgo"

// Config selects providers. Nil providers use the OpenTelemetry globals.
type Config struct {
	TracerProvider trace.TracerProvider
	MeterProvider  metric.MeterProvider
}

// Telemetry records MTGO RPC and connection observations with OpenTelemetry.
type Telemetry struct {
	tracer           trace.Tracer
	rpcDuration      metric.Float64Histogram
	rpcAttempts      metric.Int64Counter
	connectionEvents metric.Int64Counter
	floodWait        metric.Float64Histogram
}

var _ mtgo.Telemetry = (*Telemetry)(nil)

// New creates an OpenTelemetry adapter and its instruments.
func New(cfg Config) (*Telemetry, error) {
	tracerProvider := cfg.TracerProvider
	if tracerProvider == nil {
		tracerProvider = gootel.GetTracerProvider()
	}
	meterProvider := cfg.MeterProvider
	if meterProvider == nil {
		meterProvider = gootel.GetMeterProvider()
	}

	meter := meterProvider.Meter(instrumentationName)
	rpcDuration, err1 := meter.Float64Histogram("mtgo.rpc.duration", metric.WithUnit("s"), metric.WithDescription("MTProto RPC attempt latency"))
	rpcAttempts, err2 := meter.Int64Counter("mtgo.rpc.attempts", metric.WithDescription("MTProto RPC attempts"))
	connectionEvents, err3 := meter.Int64Counter("mtgo.connection.events", metric.WithDescription("MTProto connection lifecycle events"))
	floodWait, err4 := meter.Float64Histogram("mtgo.rpc.flood_wait", metric.WithUnit("s"), metric.WithDescription("Telegram flood wait duration"))
	if err := errors.Join(err1, err2, err3, err4); err != nil {
		return nil, err
	}
	return &Telemetry{
		tracer:           tracerProvider.Tracer(instrumentationName),
		rpcDuration:      rpcDuration,
		rpcAttempts:      rpcAttempts,
		connectionEvents: connectionEvents,
		floodWait:        floodWait,
	}, nil
}

// ObserveRPC records one physical MTProto attempt.
func (t *Telemetry) ObserveRPC(ctx context.Context, observation mtgo.RPCObservation) {
	attrs := []attribute.KeyValue{
		attribute.String("rpc.system", "telegram.mtproto"),
		attribute.String("rpc.method", observation.Method),
		attribute.Int("server.dc", observation.DCID),
		attribute.Int("rpc.attempt", observation.Attempt),
		attribute.String("error.type", observation.ErrorClass),
	}
	if observation.DeliveryState != "" {
		attrs = append(attrs, attribute.String("mtgo.delivery.state", string(observation.DeliveryState)))
	}
	set := attribute.NewSet(attrs...)
	metricOptions := metric.WithAttributeSet(set)
	t.rpcAttempts.Add(ctx, 1, metricOptions)
	t.rpcDuration.Record(ctx, observation.EndedAt.Sub(observation.StartedAt).Seconds(), metricOptions)
	if observation.FloodWait > 0 {
		t.floodWait.Record(ctx, observation.FloodWait.Seconds(), metricOptions)
	}

	_, span := t.tracer.Start(ctx, "mtproto."+observation.Method,
		trace.WithSpanKind(trace.SpanKindClient),
		trace.WithTimestamp(observation.StartedAt),
		trace.WithAttributes(attrs...),
	)
	if observation.Error != nil {
		span.RecordError(observation.Error)
		span.SetStatus(codes.Error, observation.ErrorClass)
	} else {
		span.SetStatus(codes.Ok, "")
	}
	span.End(trace.WithTimestamp(observation.EndedAt))
}

// ObserveConnection records one transport lifecycle event.
func (t *Telemetry) ObserveConnection(ctx context.Context, observation mtgo.ConnectionObservation) {
	attrs := []attribute.KeyValue{
		attribute.String("mtgo.connection.event", observation.Kind),
		attribute.String("server.address", observation.Endpoint),
		attribute.Bool("error.present", observation.Error != nil),
	}
	set := attribute.NewSet(attrs...)
	t.connectionEvents.Add(ctx, 1, metric.WithAttributeSet(set))

	_, span := t.tracer.Start(ctx, "mtproto.connection."+observation.Kind,
		trace.WithSpanKind(trace.SpanKindClient),
		trace.WithTimestamp(observation.StartedAt),
		trace.WithAttributes(attrs...),
	)
	if observation.Error != nil {
		span.RecordError(observation.Error)
		span.SetStatus(codes.Error, observation.Error.Error())
	}
	span.End(trace.WithTimestamp(observation.EndedAt))
}
