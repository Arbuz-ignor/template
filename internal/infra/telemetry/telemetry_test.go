package telemetry_test

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"go.opentelemetry.io/otel/trace"

	"github.com/Arbuz-ignor/template/internal/infra/telemetry"
)

const ratioSampler = "traceidratio"

func TestSampler(t *testing.T) {
	for _, name := range []string{"always_on", "always_off", "parentbased_always_on", "parentbased_always_off", ratioSampler, "parentbased_traceidratio"} {
		if _, err := telemetry.Sampler(name, "0.5"); err != nil {
			t.Fatal(err)
		}
	}
	for _, tc := range []struct{ name, arg string }{{"unknown", ""}, {ratioSampler, "NaN"}, {ratioSampler, "1.1"}} {
		if _, err := telemetry.Sampler(tc.name, tc.arg); err == nil {
			t.Fatalf("accepted %+v", tc)
		}
	}
}

func TestLogCorrelation(t *testing.T) {
	var buffer bytes.Buffer
	logger := telemetry.NewLogger(slog.NewJSONHandler(&buffer, nil))
	traceID, _ := trace.TraceIDFromHex("4bf92f3577b34da6a3ce929d0e0e4736")
	spanID, _ := trace.SpanIDFromHex("00f067aa0ba902b7")
	ctx := trace.ContextWithSpanContext(context.Background(), trace.NewSpanContext(trace.SpanContextConfig{TraceID: traceID, SpanID: spanID}))
	logger.With("service", "trip-service").InfoContext(ctx, "trip created")
	var record map[string]any
	if err := json.Unmarshal(buffer.Bytes(), &record); err != nil {
		t.Fatal(err)
	}
	if record["trace_id"] != traceID.String() || record["span_id"] != spanID.String() {
		t.Fatalf("log: %+v", record)
	}
}

func TestOTLPExport(t *testing.T) {
	var mu sync.Mutex
	paths := map[string]int{}
	collector := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		paths[r.URL.Path]++
		mu.Unlock()
		w.Header().Set("Content-Type", "application/x-protobuf")
		w.WriteHeader(200)
	}))
	defer collector.Close()
	t.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", collector.URL)
	t.Setenv("OTEL_EXPORTER_OTLP_PROTOCOL", "http/protobuf")
	t.Setenv("OTEL_TRACES_SAMPLER", "always_on")
	ctx := context.Background()
	signals, err := telemetry.New(ctx, slog.LevelInfo)
	if err != nil {
		t.Fatal(err)
	}
	ctx, span := signals.Traces.Tracer("test").Start(ctx, "request")
	business, err := telemetry.NewBusiness(signals.Meters.Meter("test"))
	if err != nil {
		t.Fatal(err)
	}
	business.Created(ctx)
	business.Completed(ctx)
	span.End()
	if err := signals.Shutdown(ctx); err != nil {
		t.Fatal(err)
	}
	mu.Lock()
	defer mu.Unlock()
	if paths["/v1/traces"] == 0 || paths["/v1/metrics"] == 0 {
		t.Fatalf("exports: %+v", paths)
	}
}
