package create

import (
	"bytes"
	"encoding/json"
	"github.com/Arbuz-ignor/template/internal/domain/trip"
	api "github.com/Arbuz-ignor/template/internal/generated"
	"github.com/google/uuid"
	"io"
	"net/http"
)

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
		return data, &trip.ValidationError{Field: "body", Reason: "extra JSON value"}
	}
	var fields map[string]json.RawMessage
	if json.Unmarshal(raw, &fields) != nil || !hasFields(fields, "user_id", "driver_id", "start_point", "end_point", "price") {
		return data, &trip.ValidationError{Field: "body", Reason: "missing field"}
	}
	for _, point := range []string{"start_point", "end_point"} {
		var coords map[string]json.RawMessage
		if json.Unmarshal(fields[point], &coords) != nil || !hasFields(coords, "latitude", "longitude") {
			return data, &trip.ValidationError{Field: point, Reason: "missing coordinate"}
		}
	}
	if data.UserId == uuid.Nil || data.DriverId == uuid.Nil || data.Price < 0 ||
		data.StartPoint.Latitude < -90 || data.StartPoint.Latitude > 90 ||
		data.EndPoint.Latitude < -90 || data.EndPoint.Latitude > 90 ||
		data.StartPoint.Longitude < -180 || data.StartPoint.Longitude > 180 ||
		data.EndPoint.Longitude < -180 || data.EndPoint.Longitude > 180 {
		return data, &trip.ValidationError{Field: "body", Reason: "invalid field"}
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
