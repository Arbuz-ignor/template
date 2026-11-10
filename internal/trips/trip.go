package trips

import "github.com/Arbuz-ignor/template/internal/domain/trip"

type Trip = trip.Trip
type Coordinates = trip.Coordinates

var (
	ErrNotFound   = trip.ErrNotFound
	ErrCompleted  = trip.ErrCompleted
	ErrDriverBusy = trip.ErrDriverBusy
)
