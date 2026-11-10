package create

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

func (u *Usecase) Execute(ctx context.Context, input trip.Trip) (trip.Trip, error) {
	if err := input.Validate(); err != nil {
		return trip.Trip{}, err
	}
	id, err := uuid.NewRandom()
	if err != nil {
		return trip.Trip{}, fmt.Errorf("generate trip id: %w", err)
	}
	input.ID = id
	input.Status = trip.Active
	input.StartedAt = time.Now().UTC().Truncate(time.Microsecond)
	err = u.tx.Do(ctx, func(ctx context.Context) error {
		if err := u.repo.Create(ctx, input); err != nil {
			return err
		}
		return u.repo.AddStatus(ctx, input.ID, nil, trip.Active, "trip created")
	})
	if err != nil {
		return trip.Trip{}, fmt.Errorf("create trip: %w", err)
	}
	u.events.Created(ctx)
	u.logger.InfoContext(ctx, "trip created", "trip_id", input.ID.String())
	return input, nil
}
