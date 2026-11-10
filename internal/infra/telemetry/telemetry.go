package telemetry

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strconv"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/metric"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
)

type Telemetry struct {
	Traces *sdktrace.TracerProvider
	Meters *sdkmetric.MeterProvider
	Logger *slog.Logger
}

func New(ctx context.Context, level slog.Level) (*Telemetry, error) {
	res, err := resource.New(ctx, resource.WithFromEnv(), resource.WithAttributes(attribute.String("service.name", envDefault("OTEL_SERVICE_NAME", "trip-service")), attribute.String("service.version", envDefault("SERVICE_VERSION", "local")), attribute.String("service.namespace", "tripgo"), attribute.String("deployment.environment", "local")))
	if err != nil {
		return nil, fmt.Errorf("telemetry resource: %w", err)
	}
	sampler, err := Sampler(os.Getenv("OTEL_TRACES_SAMPLER"), os.Getenv("OTEL_TRACES_SAMPLER_ARG"))
	if err != nil {
		return nil, err
	}
	spanExporter, metricExporter, err := exporters(ctx, os.Getenv("OTEL_EXPORTER_OTLP_PROTOCOL"))
	if err != nil {
		return nil, err
	}
	traces := sdktrace.NewTracerProvider(sdktrace.WithResource(res), sdktrace.WithSampler(sampler), sdktrace.WithBatcher(spanExporter))
	meters := sdkmetric.NewMeterProvider(sdkmetric.WithResource(res), sdkmetric.WithReader(sdkmetric.NewPeriodicReader(metricExporter)))
	logger := slog.New(&traceHandler{next: slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: level})}).With("service", "trip-service")
	return &Telemetry{Traces: traces, Meters: meters, Logger: logger}, nil
}

func exporters(ctx context.Context, protocol string) (sdktrace.SpanExporter, sdkmetric.Exporter, error) {
	switch protocol {
	case "http/protobuf", "":
		traces, err := otlptracehttp.New(ctx)
		if err != nil {
			return nil, nil, err
		}
		metrics, err := otlpmetrichttp.New(ctx)
		if err != nil {
			return nil, nil, errors.Join(err, traces.Shutdown(ctx))
		}
		return traces, metrics, nil
	case "grpc":
		traces, err := otlptracegrpc.New(ctx)
		if err != nil {
			return nil, nil, err
		}
		metrics, err := otlpmetricgrpc.New(ctx)
		if err != nil {
			return nil, nil, errors.Join(err, traces.Shutdown(ctx))
		}
		return traces, metrics, nil
	default:
		return nil, nil, fmt.Errorf("unsupported OTEL_EXPORTER_OTLP_PROTOCOL: %s", protocol)
	}
}

func (t *Telemetry) Shutdown(ctx context.Context) error {
	return errors.Join(t.Meters.Shutdown(ctx), t.Traces.Shutdown(ctx))
}

func Sampler(name, arg string) (sdktrace.Sampler, error) {
	switch name {
	case "always_on":
		return sdktrace.AlwaysSample(), nil
	case "always_off":
		return sdktrace.NeverSample(), nil
	case "parentbased_always_off":
		return sdktrace.ParentBased(sdktrace.NeverSample()), nil
	case "", "parentbased_always_on":
		return sdktrace.ParentBased(sdktrace.AlwaysSample()), nil
	case "traceidratio", "parentbased_traceidratio":
		ratio, err := strconv.ParseFloat(arg, 64)
		if err != nil || !(ratio >= 0 && ratio <= 1) {
			return nil, fmt.Errorf("OTEL_TRACES_SAMPLER_ARG must be between 0 and 1")
		}
		sampler := sdktrace.TraceIDRatioBased(ratio)
		if name == "parentbased_traceidratio" {
			return sdktrace.ParentBased(sampler), nil
		}
		return sampler, nil
	default:
		return nil, fmt.Errorf("unsupported OTEL_TRACES_SAMPLER: %s", name)
	}
}

func envDefault(name, fallback string) string {
	if value := os.Getenv(name); value != "" {
		return value
	}
	return fallback
}

type traceHandler struct{ next slog.Handler }

func (h *traceHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.next.Enabled(ctx, level)
}

func (h *traceHandler) Handle(ctx context.Context, record slog.Record) error {
	span := trace.SpanContextFromContext(ctx)
	traceID, spanID := "", ""
	if span.IsValid() {
		traceID = span.TraceID().String()
		spanID = span.SpanID().String()
	}
	record.AddAttrs(slog.String("trace_id", traceID), slog.String("span_id", spanID))
	return h.next.Handle(ctx, record)
}

func (h *traceHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &traceHandler{next: h.next.WithAttrs(attrs)}
}

func (h *traceHandler) WithGroup(name string) slog.Handler {
	return &traceHandler{next: h.next.WithGroup(name)}
}
func NewLogger(handler slog.Handler) *slog.Logger { return slog.New(&traceHandler{next: handler}) }

type Business struct {
	created   metric.Int64Counter
	completed metric.Int64Counter
}

func NewBusiness(meter metric.Meter) (*Business, error) {
	created, err := meter.Int64Counter("tripgo.trips.created")
	if err != nil {
		return nil, err
	}
	completed, err := meter.Int64Counter("tripgo.trips.completed")
	if err != nil {
		return nil, err
	}
	return &Business{created: created, completed: completed}, nil
}
func (b *Business) Created(ctx context.Context)   { b.created.Add(ctx, 1) }
func (b *Business) Completed(ctx context.Context) { b.completed.Add(ctx, 1) }
