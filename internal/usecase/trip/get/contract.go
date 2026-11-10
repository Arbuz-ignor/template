package get

import (
	"context"
	"github.com/Arbuz-ignor/template/internal/domain/trip"
	"github.com/google/uuid"
)

type tripRepository interface {
	Get(context.Context, uuid.UUID) (trip.Trip, error)
}
