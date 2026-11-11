package middleware

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"runtime/debug"
	"strings"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"

	"github.com/Arbuz-ignor/template/internal/http/domainerror"
)

type Middleware struct {
	logger   *slog.Logger
	tracer   trace.Tracer
	duration metric.Float64Histogram
	active   metric.Int64UpDownCounter
}

func New(logger *slog.Logger, tp trace.TracerProvider, mp metric.MeterProvider) (*Middleware, error) {
	meter := mp.Meter("trip-service/http")
	duration, err := meter.Float64Histogram("http.server.request.duration", metric.WithUnit("s"))
	if err != nil {
		return nil, err
	}
	active, err := meter.Int64UpDownCounter("http.server.active_requests")
	if err != nil {
		return nil, err
	}
	return &Middleware{logger: logger, tracer: tp.Tracer("trip-service/http"), duration: duration, active: active}, nil
}

type writer struct {
	http.ResponseWriter
	status int
}

func (w *writer) WriteHeader(status int) {
	if w.status != 0 {
		return
	}
	w.status = status
	w.ResponseWriter.WriteHeader(status)
}

func (w *writer) Write(data []byte) (int, error) {
	if w.status == 0 {
		w.WriteHeader(http.StatusOK)
	}
	return w.ResponseWriter.Write(data)
}
func (w *writer) Unwrap() http.ResponseWriter { return w.ResponseWriter }
func (w *writer) code() int {
	if w.status == 0 {
		return http.StatusOK
	}
	return w.status
}

type (
	requestKey   struct{}
	requestState struct {
		ctx    context.Context
		writer *writer
	}
)

func (m *Middleware) Recover(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		state := &requestState{ctx: r.Context(), writer: &writer{ResponseWriter: w}}
		r = r.WithContext(context.WithValue(r.Context(), requestKey{}, state))
		defer func() {
			if problem := recover(); problem != nil {
				m.logger.ErrorContext(state.ctx, "panic recovered", "error", fmt.Sprint(problem), "stack", string(debug.Stack()))
				domainerror.Write(state.writer, r, fmt.Errorf("handler panic: %v", problem))
			}
		}()
		next.ServeHTTP(state.writer, r)
	})
}

func (m *Middleware) Tracing(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := propagation.TraceContext{}.Extract(r.Context(), propagation.HeaderCarrier(r.Header))
		route := Route(r.URL.Path)
		ctx, span := m.tracer.Start(ctx, r.Method+" "+route, trace.WithSpanKind(trace.SpanKindServer), trace.WithAttributes(attribute.String("http.route", route), attribute.String("http.request.method", r.Method)))
		state := r.Context().Value(requestKey{}).(*requestState)
		state.ctx = ctx
		defer func() {
			problem := recover()
			status := state.writer.code()
			if problem != nil {
				status = 500
				span.RecordError(fmt.Errorf("handler panic: %v", problem))
				span.SetAttributes(attribute.String("error.type", "panic"))
			}
			span.SetAttributes(attribute.Int("http.response.status_code", status))
			if status >= 500 {
				span.SetStatus(codes.Error, "request failed")
				if problem == nil {
					span.SetAttributes(attribute.String("error.type", "internal_error"))
				}
			}
			span.End()
			if problem != nil {
				panic(problem)
			}
		}()
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (m *Middleware) Logging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		var requestErr error
		r = r.WithContext(domainerror.WithErrorSink(r.Context(), func(err error) { requestErr = err }))
		state := r.Context().Value(requestKey{}).(*requestState)
		defer func() {
			problem := recover()
			status := state.writer.code()
			if problem != nil {
				status = 500
			}
			level := slog.LevelInfo
			if status >= 500 {
				level = slog.LevelError
			}
			m.logger.Log(r.Context(), level, "request handled", "method", r.Method, "route", Route(r.URL.Path), "status", status, "duration_ms", float64(time.Since(start).Microseconds())/1000, "error", requestErr)
			if problem != nil {
				panic(problem)
			}
		}()
		next.ServeHTTP(w, r)
	})
}

func (m *Middleware) Metrics(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		state := r.Context().Value(requestKey{}).(*requestState)
		attrs := []attribute.KeyValue{attribute.String("http.request.method", r.Method), attribute.String("http.route", Route(r.URL.Path))}
		m.active.Add(r.Context(), 1, metric.WithAttributes(attrs...))
		defer func() {
			problem := recover()
			status := state.writer.code()
			if problem != nil {
				status = 500
			}
			m.active.Add(r.Context(), -1, metric.WithAttributes(attrs...))
			m.duration.Record(r.Context(), time.Since(start).Seconds(), metric.WithAttributes(append(attrs, attribute.Int("http.response.status_code", status))...))
			if problem != nil {
				panic(problem)
			}
		}()
		next.ServeHTTP(w, r)
	})
}

func Route(path string) string {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) == 1 && (path == "/health" || path == "/ready") {
		return path
	}
	if len(parts) >= 3 && parts[0] == "api" && parts[1] == "v1" && parts[2] == "trips" {
		if len(parts) == 3 {
			return "/api/v1/trips"
		}
		if len(parts) == 4 {
			return "/api/v1/trips/{tripId}"
		}
		if len(parts) == 5 && parts[4] == "finish" {
			return "/api/v1/trips/{tripId}/finish"
		}
	}
	return "unmatched"
}
