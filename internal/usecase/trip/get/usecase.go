package get

import (
	"context"
	"fmt"
	"github.com/Arbuz-ignor/template/internal/domain/trip"
	"github.com/google/uuid"
)

type Usecase struct{ repo tripRepository }

func New(repo tripRepository) *Usecase { return &Usecase{repo: repo} }
func (u *Usecase) Execute(ctx context.Context, id uuid.UUID) (trip.Trip, error) {
	result, err := u.repo.Get(ctx, id)
	if err != nil {
		return trip.Trip{}, fmt.Errorf("get trip: %w", err)
	}
	return result, nil
}
