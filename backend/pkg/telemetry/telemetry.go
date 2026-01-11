package telemetry

import (
	"context"
	"log/slog"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"
	"go.opentelemetry.io/otel/trace"
)

// Config holds telemetry configuration
type Config struct {
	ServiceName    string
	ServiceVersion string
	Environment    string
	OTLPEndpoint   string
	Enabled        bool
}

// Provider wraps OpenTelemetry providers
type Provider struct {
	tracerProvider *sdktrace.TracerProvider
	tracer         trace.Tracer
	enabled        bool
}

// NewProvider creates a new telemetry provider
func NewProvider(ctx context.Context, cfg *Config) (*Provider, error) {
	if !cfg.Enabled {
		slog.Info("Telemetry disabled")
		return &Provider{enabled: false}, nil
	}

	// Create resource
	res, err := resource.Merge(
		resource.Default(),
		resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceName(cfg.ServiceName),
			semconv.ServiceVersion(cfg.ServiceVersion),
			attribute.String("environment", cfg.Environment),
		),
	)
	if err != nil {
		return nil, err
	}

	// Create OTLP exporter
	exporter, err := otlptracegrpc.New(ctx,
		otlptracegrpc.WithEndpoint(cfg.OTLPEndpoint),
		otlptracegrpc.WithInsecure(),
	)
	if err != nil {
		return nil, err
	}

	// Create tracer provider
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter,
			sdktrace.WithBatchTimeout(5*time.Second),
		),
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
	)

	// Set global tracer provider
	otel.SetTracerProvider(tp)

	// Set global propagator
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	slog.Info("Telemetry initialized",
		"service", cfg.ServiceName,
		"endpoint", cfg.OTLPEndpoint,
	)

	return &Provider{
		tracerProvider: tp,
		tracer:         tp.Tracer(cfg.ServiceName),
		enabled:        true,
	}, nil
}

// Shutdown gracefully shuts down the telemetry provider
func (p *Provider) Shutdown(ctx context.Context) error {
	if !p.enabled || p.tracerProvider == nil {
		return nil
	}
	return p.tracerProvider.Shutdown(ctx)
}

// Tracer returns the tracer instance
func (p *Provider) Tracer() trace.Tracer {
	if !p.enabled {
		return otel.Tracer("noop")
	}
	return p.tracer
}

// IsEnabled returns whether telemetry is enabled
func (p *Provider) IsEnabled() bool {
	return p.enabled
}

// StartSpan starts a new span with the given name
func (p *Provider) StartSpan(ctx context.Context, name string, opts ...trace.SpanStartOption) (context.Context, trace.Span) {
	return p.tracer.Start(ctx, name, opts...)
}

// SpanFromContext returns the current span from context
func SpanFromContext(ctx context.Context) trace.Span {
	return trace.SpanFromContext(ctx)
}

// AddSpanAttributes adds attributes to the current span
func AddSpanAttributes(ctx context.Context, attrs ...attribute.KeyValue) {
	span := trace.SpanFromContext(ctx)
	span.SetAttributes(attrs...)
}

// RecordError records an error on the current span
func RecordError(ctx context.Context, err error) {
	span := trace.SpanFromContext(ctx)
	span.RecordError(err)
}
