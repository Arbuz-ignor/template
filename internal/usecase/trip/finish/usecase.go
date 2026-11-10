package finish

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"github.com/Arbuz-ignor/template/internal/domain/trip"
)

type Usecase struct {
	logger *slog.Logger
	events eventRecorder
	repo   tripRepository
	tx     txManager
}

func New(repo tripRepository, tx txManager, logger *slog.Logger, events eventRecorder) *Usecase {
	return &Usecase{repo: repo, tx: tx, logger: logger, events: events}
}

func (u *Usecase) Execute(ctx context.Context, id uuid.UUID) (trip.Trip, error) {
	var result trip.Trip
	err := u.tx.Do(ctx, func(ctx context.Context) error {
		var err error
		result, err = u.repo.GetForUpdate(ctx, id)
		if err != nil {
			return err
		}
		if err = result.Finish(time.Now().UTC().Truncate(time.Microsecond)); err != nil {
			return err
		}
		if err = u.repo.SaveCompleted(ctx, result); err != nil {
			return err
		}
		from := trip.Active
		return u.repo.AddStatus(ctx, id, &from, trip.Completed, "trip finished")
	})
	if err != nil {
		return trip.Trip{}, fmt.Errorf("finish trip: %w", err)
	}
	u.events.Completed(ctx)
	u.logger.InfoContext(ctx, "trip completed", "trip_id", result.ID.String())
	return result, nil
}
