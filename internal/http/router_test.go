package httpserver_test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"go.uber.org/mock/gomock"

	httpserver "github.com/Arbuz-ignor/template/internal/http"
)

const readyPath = "/ready"

func TestHealthAndReady(t *testing.T) {
	for _, tc := range []struct {
		name, path string
		dbErr      error
		status     int
	}{{"health", "/health", nil, 200}, {"ready", readyPath, nil, 200}, {"unavailable", readyPath, errors.New("offline"), 503}} {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			db := NewMockpinger(ctrl)
			if tc.path == readyPath {
				db.EXPECT().Ping(gomock.Any()).Return(tc.dbErr)
			}
			w := httptest.NewRecorder()
			httpserver.Router(db, httpserver.Handlers{}, time.Second).ServeHTTP(w, httptest.NewRequest(http.MethodGet, tc.path, nil))
			if w.Code != tc.status {
				t.Fatalf("response: %d %s", w.Code, w.Body)
			}
		})
	}
}

func TestInvalidTripID(t *testing.T) {
	ctrl := gomock.NewController(t)
	db := NewMockpinger(ctrl)
	w := httptest.NewRecorder()
	httpserver.Router(db, httpserver.Handlers{}, time.Second).ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/v1/trips/invalid", nil))
	if w.Code != 400 {
		t.Fatalf("response: %d %s", w.Code, w.Body)
	}
}
