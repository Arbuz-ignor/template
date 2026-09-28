package httpserver

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

type fakeDB struct{ err error }

func (f fakeDB) Ping(context.Context) error { return f.err }

func TestHealthIsLiveWhenDatabaseIsUnavailable(t *testing.T) {
	router := Router(fakeDB{err: errors.New("database unavailable")}, time.Second)
	for _, tc := range []struct {
		path string
		code int
		body string
	}{
		{"/health", http.StatusOK, `{"status":"ok"}` + "\n"},
		{"/ready", http.StatusServiceUnavailable, `{"status":"unavailable"}` + "\n"},
	} {
		response := httptest.NewRecorder()
		router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, tc.path, nil))
		if response.Code != tc.code || response.Body.String() != tc.body ||
			response.Header().Get("Content-Type") != "application/json" {
			t.Errorf("%s: code=%d body=%q headers=%v", tc.path, response.Code, response.Body.String(), response.Header())
		}
	}
}

func TestReadyWhenDatabaseResponds(t *testing.T) {
	response := httptest.NewRecorder()
	Router(fakeDB{}, time.Second).ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/ready", nil))
	if response.Code != http.StatusOK || response.Body.String() != `{"status":"ok"}`+"\n" {
		t.Fatalf("unexpected readiness response: code=%d body=%q", response.Code, response.Body.String())
	}
}
