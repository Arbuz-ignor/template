//go:generate go tool mockgen -source=contract.go -destination=mocks_test.go -package=create_test

package create

import (
	"context"

	"github.com/Arbuz-ignor/template/internal/domain/trip"
)

type usecase interface {
	Execute(context.Context, trip.Trip) (trip.Trip, error)
}
