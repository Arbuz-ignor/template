//go:generate go tool mockgen -source=contract.go -destination=mocks_test.go -package=get_test

package get

import (
	"context"
	"github.com/Arbuz-ignor/template/internal/domain/trip"
	"github.com/google/uuid"
)

type usecase interface {
	Execute(context.Context, uuid.UUID) (trip.Trip, error)
}
