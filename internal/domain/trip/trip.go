package trip

import (
	"errors"
	"fmt"
	"github.com/google/uuid"
	"time"
)

const (
	Active    = "active"
	Completed = "completed"
)

var (
	ErrNotFound   = errors.New("trip not found")
	ErrCompleted  = errors.New("trip already completed")
	ErrDriverBusy = errors.New("driver already has an active trip")
)

type ValidationError struct {
	Field  string
	Reason string
}

func (e *ValidationError) Error() string { return fmt.Sprintf("%s: %s", e.Field, e.Reason) }

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

func (t Trip) Validate() error {
	if t.UserID == uuid.Nil {
		return &ValidationError{Field: "user_id", Reason: "must be a nonzero UUID"}
	}
	if t.DriverID == uuid.Nil {
		return &ValidationError{Field: "driver_id", Reason: "must be a nonzero UUID"}
	}
	if t.Price < 0 {
		return &ValidationError{Field: "price", Reason: "must be nonnegative"}
	}
	for _, point := range []struct {
		name  string
		value Coordinates
	}{{"start_point", t.StartPoint}, {"end_point", t.EndPoint}} {
		if !(point.value.Latitude >= -90 && point.value.Latitude <= 90) {
			return &ValidationError{Field: point.name + ".latitude", Reason: "must be between -90 and 90"}
		}
		if !(point.value.Longitude >= -180 && point.value.Longitude <= 180) {
			return &ValidationError{Field: point.name + ".longitude", Reason: "must be between -180 and 180"}
		}
	}
	return nil
}

func (t *Trip) Finish(at time.Time) error {
	if t.Status != Active {
		return ErrCompleted
	}
	t.Status = Completed
	at = at.UTC()
	t.FinishedAt = &at
	return nil
}
