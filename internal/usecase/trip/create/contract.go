//go:generate go tool mockgen -source=contract.go -destination=mocks_test.go -package=create_test

package create

import (
	"context"
	"github.com/Arbuz-ignor/template/internal/domain/trip"
	"github.com/google/uuid"
)

type tripRepository interface {
	Create(context.Context, trip.Trip) error
	AddStatus(context.Context, uuid.UUID, *string, string, string) error
}
type txManager interface {
	Do(context.Context, func(context.Context) error) error
}
