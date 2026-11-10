package response

import (
	"encoding/json"
	"net/http"

	"github.com/Arbuz-ignor/template/internal/domain/trip"
	api "github.com/Arbuz-ignor/template/internal/generated"
)

func JSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func Trip(t trip.Trip) api.Trip {
	return api.Trip{Id: t.ID, UserId: t.UserID, DriverId: t.DriverID, StartPoint: api.Coordinates{Latitude: t.StartPoint.Latitude, Longitude: t.StartPoint.Longitude}, EndPoint: api.Coordinates{Latitude: t.EndPoint.Latitude, Longitude: t.EndPoint.Longitude}, Price: t.Price, Status: api.TripStatus(t.Status), StartedAt: t.StartedAt, FinishedAt: t.FinishedAt}
}
