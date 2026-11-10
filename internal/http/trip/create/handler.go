package create

import (
	"mime"
	"net/http"

	"github.com/Arbuz-ignor/template/internal/domain/trip"
	"github.com/Arbuz-ignor/template/internal/http/domainerror"
	"github.com/Arbuz-ignor/template/internal/http/response"
)

type Handler struct{ usecase usecase }

func New(u usecase) *Handler { return &Handler{usecase: u} }
func (h *Handler) Handle(w http.ResponseWriter, r *http.Request) {
	media, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || media != "application/json" {
		domainerror.Write(w, r, &trip.ValidationError{Field: bodyField, Reason: "expected application/json"})
		return
	}
	data, err := readTripData(w, r)
	if err != nil {
		domainerror.Write(w, r, &trip.ValidationError{Field: bodyField, Reason: "invalid trip data"})
		return
	}
	result, err := h.usecase.Execute(r.Context(), trip.Trip{UserID: data.UserId, DriverID: data.DriverId, StartPoint: trip.Coordinates{Latitude: data.StartPoint.Latitude, Longitude: data.StartPoint.Longitude}, EndPoint: trip.Coordinates{Latitude: data.EndPoint.Latitude, Longitude: data.EndPoint.Longitude}, Price: data.Price})
	if err != nil {
		domainerror.Write(w, r, err)
		return
	}
	w.Header().Set("Location", "/api/v1/trips/"+result.ID.String())
	response.JSON(w, http.StatusCreated, response.Trip(result))
}
