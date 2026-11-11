package middleware_test

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"

	"github.com/Arbuz-ignor/template/internal/http/middleware"
)

func TestRequestsAndPanic(t *testing.T) {
	for _, panics := range []bool{false, true} {
		t.Run(map[bool]string{false: "normal", true: "panic"}[panics], func(t *testing.T) {
			var logs bytes.Buffer
			exporter := tracetest.NewInMemoryExporter()
			tp := sdktrace.NewTracerProvider(sdktrace.WithSyncer(exporter))
			reader := metric.NewManualReader()
			mp := metric.NewMeterProvider(metric.WithReader(reader))
			t.Cleanup(func() { _ = tp.Shutdown(context.Background()); _ = mp.Shutdown(context.Background()) })
			m, err := middleware.New(slog.New(slog.NewJSONHandler(&logs, nil)), tp, mp)
			if err != nil {
				t.Fatal(err)
			}
			h := m.Recover(m.Tracing(m.Logging(m.Metrics(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				if panics {
					panic("broken handler")
				}
				w.WriteHeader(201)
			})))))
			r := httptest.NewRequest(http.MethodPost, "/api/v1/trips/123/finish", nil)
			r.Header.Set("traceparent", "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01")
			w := httptest.NewRecorder()
			h.ServeHTTP(w, r)
			want := 201
			if panics {
				want = 500
			}
			if w.Code != want {
				t.Fatalf("response: %d %s", w.Code, w.Body)
			}
			spans := exporter.GetSpans()
			if len(spans) != 1 || spans[0].Name != "POST /api/v1/trips/{tripId}/finish" || spans[0].SpanContext.TraceID().String() != "4bf92f3577b34da6a3ce929d0e0e4736" {
				t.Fatalf("spans: %+v", spans)
			}
			var record map[string]any
			if err := json.NewDecoder(&logs).Decode(&record); err != nil {
				t.Fatal(err)
			}
			if record["status"] != float64(want) {
				t.Fatalf("log: %+v", record)
			}
			var data metricdata.ResourceMetrics
			if err := reader.Collect(context.Background(), &data); err != nil {
				t.Fatal(err)
			}
			seen := 0
			for _, scope := range data.ScopeMetrics {
				for _, value := range scope.Metrics {
					switch points := value.Data.(type) {
					case metricdata.Sum[int64]:
						for _, point := range points.DataPoints {
							if point.Value != 0 {
								t.Fatalf("active requests leaked: %d", point.Value)
							}
							seen++
						}
					case metricdata.Histogram[float64]:
						for _, point := range points.DataPoints {
							if point.Count != 1 {
								t.Fatalf("request count: %d", point.Count)
							}
							seen++
						}
					}
				}
			}
			if seen != 2 {
				t.Fatalf("metrics missing: %+v", data)
			}
		})
	}
}

func TestRoute(t *testing.T) {
	for _, tc := range []struct{ path, want string }{{"/health", "/health"}, {"/ready", "/ready"}, {"/api/v1/trips", "/api/v1/trips"}, {"/api/v1/trips/123", "/api/v1/trips/{tripId}"}, {"/unknown/123", "unmatched"}} {
		if got := middleware.Route(tc.path); got != tc.want {
			t.Fatalf("%s: %s", tc.path, got)
		}
	}
}
