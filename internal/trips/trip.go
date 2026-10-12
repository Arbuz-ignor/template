package trips

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

var (
	ErrNotFound   = errors.New("trip not found")
	ErrDriverBusy = errors.New("driver already has an active trip")
	ErrCompleted  = errors.New("trip already completed")
)

type Coordinates struct {
	Latitude  float64
	Longitude float64
}

type Trip struct {
	ID         uuid.UUID
	UserID     uuid.UUID
	DriverID   uuid.UUID
	StartPoint Coordinates
	EndPoint   Coordinates
	Price      int64
	Status     string
	StartedAt  time.Time
	FinishedAt *time.Time
}
