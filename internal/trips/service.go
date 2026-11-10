package trips

import (
	"context"
	"github.com/Arbuz-ignor/template/internal/usecase/trip/create"
	"github.com/Arbuz-ignor/template/internal/usecase/trip/finish"
	"github.com/Arbuz-ignor/template/internal/usecase/trip/get"
	"github.com/google/uuid"
)

type Service struct {
	create *create.Usecase
	get    *get.Usecase
	finish *finish.Usecase
}

func NewService(repo *Repository, tx TxManager) *Service {
	return &Service{create: create.New(repo, tx), get: get.New(repo), finish: finish.New(repo, tx)}
}

type TxManager interface {
	Do(context.Context, func(context.Context) error) error
}

func (s *Service) Create(ctx context.Context, t Trip) (Trip, error)    { return s.create.Execute(ctx, t) }
func (s *Service) Get(ctx context.Context, id uuid.UUID) (Trip, error) { return s.get.Execute(ctx, id) }
func (s *Service) Finish(ctx context.Context, id uuid.UUID) (Trip, error) {
	return s.finish.Execute(ctx, id)
}
