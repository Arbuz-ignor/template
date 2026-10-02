package httpserver

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/Arbuz-ignor/template/internal/generated"
	"github.com/go-chi/chi/v5"
)

type Pinger interface {
	Ping(context.Context) error
}

func Router(db Pinger, queryTimeout time.Duration) http.Handler {
	r := chi.NewRouter()
	r.Get("/health", func(w http.ResponseWriter, _ *http.Request) {
		writeHealth(w, http.StatusOK, api.HealthResponse{Status: api.Ok})
	})
	r.Get("/ready", func(w http.ResponseWriter, request *http.Request) {
		ctx, cancel := context.WithTimeout(request.Context(), queryTimeout)
		defer cancel()
		if err := db.Ping(ctx); err != nil {
			writeHealth(w, http.StatusServiceUnavailable, api.HealthResponse{Status: api.Unavailable})
			return
		}
		writeHealth(w, http.StatusOK, api.HealthResponse{Status: api.Ok})
	})
	return r
}

func writeHealth(w http.ResponseWriter, status int, response api.HealthResponse) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(response)
}
