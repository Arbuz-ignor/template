package finish

import (
	"context"
	"fmt"
	"github.com/Arbuz-ignor/template/internal/domain/trip"
	"github.com/google/uuid"
	"time"
)

type Usecase struct {
	repo tripRepository
	tx   txManager
}

func New(repo tripRepository, tx txManager) *Usecase { return &Usecase{repo: repo, tx: tx} }
func (u *Usecase) Execute(ctx context.Context, id uuid.UUID) (trip.Trip, error) {
	var result trip.Trip
	err := u.tx.Do(ctx, func(ctx context.Context) error {
		var err error
		result, err = u.repo.GetForUpdate(ctx, id)
		if err != nil {
			return err
		}
		if err = result.Finish(time.Now().UTC()); err != nil {
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
	return result, nil
}
