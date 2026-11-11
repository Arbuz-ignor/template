package domainerror

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/Arbuz-ignor/template/internal/domain/trip"
	api "github.com/Arbuz-ignor/template/internal/generated"
)

func Problem(err error) (int, api.Problem) {
	status, code, title, detail := 500, "internal_error", "Internal error", "Internal server error"
	var validation *trip.ValidationError
	switch {
	case errors.Is(err, trip.ErrNotFound):
		status, code, title, detail = 404, "trip_not_found", "Trip not found", "Trip was not found"
	case errors.Is(err, trip.ErrCompleted):
		status, code, title, detail = 409, "trip_completed", "Trip completed", "Trip is already completed"
	case errors.Is(err, trip.ErrDriverBusy):
		status, code, title, detail = 409, "driver_busy", "Driver busy", "Driver already has an active trip"
	case errors.As(err, &validation):
		status, code, title, detail = 400, "invalid_request", "Invalid request", validation.Error()
	}
	return status, api.Problem{Type: "https://tripgo.example/problems/" + strings.ReplaceAll(code, "_", "-"), Title: title, Status: int32(status), Code: code, Detail: &detail}
}

type errorSinkKey struct{}

func WithErrorSink(ctx context.Context, sink func(error)) context.Context {
	return context.WithValue(ctx, errorSinkKey{}, sink)
}

func Write(w http.ResponseWriter, r *http.Request, err error) {
	status, body := Problem(err)
	if status >= 500 {
		if sink, ok := r.Context().Value(errorSinkKey{}).(func(error)); ok {
			sink(err)
		}
	}
	instance := r.URL.Path
	body.Instance = &instance
	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}
