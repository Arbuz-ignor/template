package telemetry

import (
	"context"

	"go.opentelemetry.io/otel/metric"
)

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
