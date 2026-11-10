//go:generate go tool mockgen -source=contract.go -destination=mocks_test.go -package=finish_test

package finish

import (
	"context"
	"github.com/Arbuz-ignor/template/internal/domain/trip"
	"github.com/google/uuid"
)

type usecase interface {
	Execute(context.Context, uuid.UUID) (trip.Trip, error)
}
