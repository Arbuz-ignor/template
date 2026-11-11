package telemetry

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/exaring/otelpgx"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/metric/noop"
	"go.opentelemetry.io/otel/trace"
)

type (
	queryKey  struct{}
	queryInfo struct {
		start                 time.Time
		operation, collection string
	}
)

type DatabaseTracer struct {
	inner    pgx.QueryTracer
	duration metric.Float64Histogram
}

func NewDatabaseTracer(tp trace.TracerProvider, mp metric.MeterProvider) (*DatabaseTracer, error) {
	duration, err := mp.Meter("trip-service/postgres").Float64Histogram("db.client.operation.duration", metric.WithUnit("s"))
	if err != nil {
		return nil, err
	}
	inner := otelpgx.NewTracer(otelpgx.WithTracerProvider(tp), otelpgx.WithMeterProvider(noop.NewMeterProvider()), otelpgx.WithDisableSQLStatementInAttributes(), otelpgx.WithDisableConnectionDetailsInAttributes(), otelpgx.WithDisableAcquireTracer())
	return &DatabaseTracer{inner: inner, duration: duration}, nil
}

func (t *DatabaseTracer) TraceQueryStart(ctx context.Context, conn *pgx.Conn, data pgx.TraceQueryStartData) context.Context {
	collection := ""
	for _, table := range []string{"trip_status_history", "trips"} {
		if strings.Contains(data.SQL, table) {
			collection = table
			break
		}
	}
	operation := otelpgx.SQLOperationName(ctx, data.SQL)
	ctx = t.inner.TraceQueryStart(ctx, conn, data)
	trace.SpanFromContext(ctx).SetAttributes(attribute.String("db.collection.name", collection), attribute.String("db.operation.name", operation))
	return context.WithValue(ctx, queryKey{}, queryInfo{start: time.Now(), operation: operation, collection: collection})
}

func (t *DatabaseTracer) TraceQueryEnd(ctx context.Context, conn *pgx.Conn, data pgx.TraceQueryEndData) {
	info, _ := ctx.Value(queryKey{}).(queryInfo)
	attrs := []attribute.KeyValue{attribute.String("db.operation.name", info.operation), attribute.String("db.collection.name", info.collection)}
	if data.Err != nil {
		kind := fmt.Sprintf("%T", data.Err)
		var pgErr *pgconn.PgError
		if errors.As(data.Err, &pgErr) {
			kind = pgErr.Code
		}
		attrs = append(attrs, attribute.String("error.type", kind))
		trace.SpanFromContext(ctx).SetAttributes(attribute.String("error.type", kind))
	}
	t.duration.Record(ctx, time.Since(info.start).Seconds(), metric.WithAttributes(attrs...))
	t.inner.TraceQueryEnd(ctx, conn, data)
}
