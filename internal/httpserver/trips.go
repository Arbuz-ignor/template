package httpserver

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"mime"
	"net/http"
	"strings"

	api "github.com/Arbuz-ignor/template/internal/generated"
	"github.com/Arbuz-ignor/template/internal/trips"
	"github.com/google/uuid"
)

func (s *Server) CreateTrip(w http.ResponseWriter, r *http.Request, _ api.CreateTripParams) {
	mediaType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || mediaType != "application/json" {
		writeProblem(w, r, http.StatusBadRequest, "invalid_request", "Invalid request", "Expected application/json")
		return
	}
	data, err := readTripData(w, r)
	if err != nil {
		writeProblem(w, r, http.StatusBadRequest, "invalid_request", "Invalid request", "Invalid trip data")
		return
	}
	trip, err := s.trips.Create(r.Context(), trips.Trip{
		UserID:   data.UserId,
		DriverID: data.DriverId,
		StartPoint: trips.Coordinates{
			Latitude: data.StartPoint.Latitude, Longitude: data.StartPoint.Longitude,
		},
		EndPoint: trips.Coordinates{
			Latitude: data.EndPoint.Latitude, Longitude: data.EndPoint.Longitude,
		},
		Price: data.Price,
	})
	if err != nil {
		writeTripError(w, r, err)
		return
	}
	w.Header().Set("Location", "/api/v1/trips/"+trip.ID.String())
	writeJSON(w, http.StatusCreated, tripResponse(trip))
}

func (s *Server) GetTrip(w http.ResponseWriter, r *http.Request, id api.TripId) {
	trip, err := s.trips.Get(r.Context(), id)
	if err != nil {
		writeTripError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, tripResponse(trip))
}

func (s *Server) FinishTrip(w http.ResponseWriter, r *http.Request, id api.TripId) {
	trip, err := s.trips.Finish(r.Context(), id)
	if err != nil {
		writeTripError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, tripResponse(trip))
}

func readTripData(w http.ResponseWriter, r *http.Request) (api.TripData, error) {
	raw, err := io.ReadAll(http.MaxBytesReader(w, r.Body, 1<<20))
	if err != nil {
		return api.TripData{}, err
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	var data api.TripData
	if err := decoder.Decode(&data); err != nil {
		return data, err
	}
	if decoder.Decode(new(any)) != io.EOF {
		return data, errors.New("extra JSON value")
	}
	var fields map[string]json.RawMessage
	if json.Unmarshal(raw, &fields) != nil || !hasFields(fields, "user_id", "driver_id", "start_point", "end_point", "price") {
		return data, errors.New("missing field")
	}
	for _, point := range []string{"start_point", "end_point"} {
		var coords map[string]json.RawMessage
		if json.Unmarshal(fields[point], &coords) != nil || !hasFields(coords, "latitude", "longitude") {
			return data, errors.New("missing coordinate")
		}
	}
	if data.UserId == uuid.Nil || data.DriverId == uuid.Nil || data.Price < 0 ||
		data.StartPoint.Latitude < -90 || data.StartPoint.Latitude > 90 ||
		data.EndPoint.Latitude < -90 || data.EndPoint.Latitude > 90 ||
		data.StartPoint.Longitude < -180 || data.StartPoint.Longitude > 180 ||
		data.EndPoint.Longitude < -180 || data.EndPoint.Longitude > 180 {
		return data, errors.New("invalid field")
	}
	return data, nil
}

func hasFields(fields map[string]json.RawMessage, names ...string) bool {
	for _, name := range names {
		value, ok := fields[name]
		if !ok || bytes.Equal(bytes.TrimSpace(value), []byte("null")) {
			return false
		}
	}
	return true
}

func tripResponse(t trips.Trip) api.Trip {
	return api.Trip{
		Id: t.ID, UserId: t.UserID, DriverId: t.DriverID,
		StartPoint: api.Coordinates{Latitude: t.StartPoint.Latitude, Longitude: t.StartPoint.Longitude},
		EndPoint:   api.Coordinates{Latitude: t.EndPoint.Latitude, Longitude: t.EndPoint.Longitude},
		Price:      t.Price, Status: api.TripStatus(t.Status), StartedAt: t.StartedAt, FinishedAt: t.FinishedAt,
	}
}

func writeTripError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, trips.ErrNotFound):
		writeProblem(w, r, http.StatusNotFound, "trip_not_found", "Trip not found", "Trip was not found")
	case errors.Is(err, trips.ErrCompleted):
		writeProblem(w, r, http.StatusConflict, "trip_completed", "Trip completed", "Trip is already completed")
	case errors.Is(err, trips.ErrDriverBusy):
		writeProblem(w, r, http.StatusConflict, "driver_busy", "Driver busy", "Driver already has an active trip")
	default:
		slog.Error("trip request failed", "error", err)
		writeProblem(w, r, http.StatusInternalServerError, "internal_error", "Internal error", "Internal server error")
	}
}

func writeProblem(w http.ResponseWriter, r *http.Request, status int, code, title, detail string) {
	instance := r.URL.Path
	problem := api.Problem{
		Type:  "https://tripgo.example/problems/" + strings.ReplaceAll(code, "_", "-"),
		Title: title, Status: int32(status), Code: code, Detail: &detail, Instance: &instance,
	}
	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(problem)
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
