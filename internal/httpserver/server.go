package httpserver

import (
	"context"
	"net/http"
	"time"

	api "github.com/Arbuz-ignor/template/internal/generated"
	"github.com/Arbuz-ignor/template/internal/trips"
	"github.com/google/uuid"
)

type Pinger interface {
	Ping(context.Context) error
}

type TripService interface {
	Create(context.Context, trips.Trip) (trips.Trip, error)
	Get(context.Context, uuid.UUID) (trips.Trip, error)
	Finish(context.Context, uuid.UUID) (trips.Trip, error)
}

type Server struct {
	db      Pinger
	trips   TripService
	timeout time.Duration
}

func Router(db Pinger, service TripService, queryTimeout time.Duration) http.Handler {
	s := &Server{db: db, trips: service, timeout: queryTimeout}
	return api.HandlerWithOptions(s, api.ChiServerOptions{
		ErrorHandlerFunc: func(w http.ResponseWriter, r *http.Request, _ error) {
			writeProblem(w, r, http.StatusBadRequest, "invalid_request", "Invalid request", "Invalid request parameter")
		},
	})
}

func (s *Server) Health(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, api.HealthResponse{Status: api.Ok})
}

func (s *Server) Ready(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), s.timeout)
	defer cancel()
	if err := s.db.Ping(ctx); err != nil {
		writeJSON(w, http.StatusServiceUnavailable, api.HealthResponse{Status: api.Unavailable})
		return
	}
	writeJSON(w, http.StatusOK, api.HealthResponse{Status: api.Ok})
}
