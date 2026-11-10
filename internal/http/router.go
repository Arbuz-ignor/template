package httpserver

import (
	"context"
	"github.com/Arbuz-ignor/template/internal/domain/trip"
	api "github.com/Arbuz-ignor/template/internal/generated"
	"github.com/Arbuz-ignor/template/internal/http/domainerror"
	"github.com/Arbuz-ignor/template/internal/http/response"
	"github.com/go-chi/chi/v5"
	"net/http"
	"time"
)

type Pinger interface{ Ping(context.Context) error }
type Handlers struct {
	Create func(http.ResponseWriter, *http.Request)
	Get    func(http.ResponseWriter, *http.Request, api.TripId)
	Finish func(http.ResponseWriter, *http.Request, api.TripId)
}
type server struct {
	db       Pinger
	handlers Handlers
	timeout  time.Duration
}

func Router(db Pinger, handlers Handlers, timeout time.Duration, middlewares ...func(http.Handler) http.Handler) http.Handler {
	s := &server{db: db, handlers: handlers, timeout: timeout}
	r := chi.NewRouter()
	r.Use(middlewares...)
	return api.HandlerWithOptions(s, api.ChiServerOptions{BaseRouter: r, ErrorHandlerFunc: func(w http.ResponseWriter, r *http.Request, _ error) {
		domainerror.Write(w, r, &trip.ValidationError{Field: "path", Reason: "invalid request parameter"})
	}})
}
func (s *server) CreateTrip(w http.ResponseWriter, r *http.Request, _ api.CreateTripParams) {
	s.handlers.Create(w, r)
}
func (s *server) GetTrip(w http.ResponseWriter, r *http.Request, id api.TripId) {
	s.handlers.Get(w, r, id)
}
func (s *server) FinishTrip(w http.ResponseWriter, r *http.Request, id api.TripId) {
	s.handlers.Finish(w, r, id)
}
func (s *server) Health(w http.ResponseWriter, _ *http.Request) {
	response.JSON(w, 200, api.HealthResponse{Status: api.Ok})
}
func (s *server) Ready(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), s.timeout)
	defer cancel()
	if err := s.db.Ping(ctx); err != nil {
		response.JSON(w, 503, api.HealthResponse{Status: api.Unavailable})
		return
	}
	response.JSON(w, 200, api.HealthResponse{Status: api.Ok})
}
