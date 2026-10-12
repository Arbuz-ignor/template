package trips

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
)

type TxManager interface {
	Do(context.Context, func(context.Context) error) error
}

type Service struct {
	repo *Repository
	tx   TxManager
}

func NewService(repo *Repository, tx TxManager) *Service {
	return &Service{repo: repo, tx: tx}
}

func (s *Service) Create(ctx context.Context, trip Trip) (Trip, error) {
	id, err := uuid.NewRandom()
	if err != nil {
		return Trip{}, fmt.Errorf("generate trip id: %w", err)
	}
	trip.ID = id
	trip.Status = "active"
	trip.StartedAt = time.Now().UTC()
	if err := s.tx.Do(ctx, func(ctx context.Context) error {
		if err := s.repo.Create(ctx, trip); err != nil {
			return err
		}
		return s.repo.AddStatus(ctx, trip.ID, nil, "active", "trip created")
	}); err != nil {
		return Trip{}, fmt.Errorf("create trip: %w", err)
	}
	return trip, nil
}

func (s *Service) Get(ctx context.Context, id uuid.UUID) (Trip, error) {
	return s.repo.Get(ctx, id)
}

func (s *Service) Finish(ctx context.Context, id uuid.UUID) (Trip, error) {
	var trip Trip
	err := s.tx.Do(ctx, func(ctx context.Context) error {
		var err error
		trip, err = s.repo.Finish(ctx, id, time.Now().UTC())
		if err != nil {
			return err
		}
		from := "active"
		return s.repo.AddStatus(ctx, id, &from, "completed", "trip finished")
	})
	if err != nil {
		return Trip{}, fmt.Errorf("finish trip: %w", err)
	}
	return trip, nil
}
