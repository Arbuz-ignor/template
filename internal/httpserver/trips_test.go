package httpserver

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Arbuz-ignor/template/internal/trips"
	"github.com/google/uuid"
)

type fakeTrips struct {
	trip  trips.Trip
	err   error
	calls int
}

func (f *fakeTrips) Create(_ context.Context, trip trips.Trip) (trips.Trip, error) {
	f.calls++
	trip.ID = uuid.MustParse("1f0a9c62-4a1c-4f2e-9d33-2a4bb0f0b111")
	trip.Status = "active"
	trip.StartedAt = time.Date(2026, 9, 28, 12, 0, 0, 0, time.UTC)
	return trip, f.err
}

func (f *fakeTrips) Get(context.Context, uuid.UUID) (trips.Trip, error) {
	f.calls++
	return f.trip, f.err
}

func (f *fakeTrips) Finish(context.Context, uuid.UUID) (trips.Trip, error) {
	f.calls++
	return f.trip, f.err
}

const validTrip = `{"user_id":"5cb72c04-7650-45c9-a79b-bcdba0631e0c","driver_id":"8860b315-ec86-42eb-a17c-7c163d721ff5","start_point":{"latitude":59.9398,"longitude":30.3146},"end_point":{"latitude":59.929,"longitude":30.3626},"price":1450}`

func TestCreateTripResponse(t *testing.T) {
	service := &fakeTrips{}
	r := httptest.NewRequest(http.MethodPost, "/api/v1/trips", strings.NewReader(validTrip))
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	Router(fakeDB{}, service, time.Second).ServeHTTP(w, r)
	if w.Code != http.StatusCreated || service.calls != 1 || w.Header().Get("Content-Type") != "application/json" {
		t.Fatalf("unexpected response: %d %s", w.Code, w.Body)
	}
	if w.Header().Get("Location") != "/api/v1/trips/1f0a9c62-4a1c-4f2e-9d33-2a4bb0f0b111" {
		t.Fatalf("location: %q", w.Header().Get("Location"))
	}
	var trip map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &trip); err != nil || trip["status"] != "active" || trip["price"] != float64(1450) {
		t.Fatalf("body: %s; error: %v", w.Body, err)
	}
}

func TestInvalidTripDataNeverCallsService(t *testing.T) {
	for _, body := range []string{
		`{"user_id":"5cb72c04-7650-45c9-a79b-bcdba0631e0c","driver_id":"8860b315-ec86-42eb-a17c-7c163d721ff5","start_point":{"latitude":59.9398,"longitude":30.3146},"end_point":{"latitude":59.929,"longitude":30.3626}}`,
		strings.Replace(validTrip, `"latitude":59.9398`, `"latitude":91`, 1),
		strings.Replace(validTrip, `"price":1450`, `"price":1450,"extra":true`, 1),
		strings.Replace(validTrip, `"price":1450`, `"price":-1`, 1),
		strings.Replace(validTrip, `"price":1450`, `"price":null`, 1),
		strings.Replace(validTrip, `"latitude":59.9398`, `"latitude":null`, 1),
		strings.Replace(validTrip, `"latitude":59.9398,`, ``, 1),
		strings.Replace(validTrip, `"longitude":30.3146`, `"longitude":181`, 1),
		strings.Replace(validTrip, `"longitude":30.3146`, `"longitude":30.3146,"extra":true`, 1),
		strings.Replace(validTrip, `5cb72c04-7650-45c9-a79b-bcdba0631e0c`, `00000000-0000-0000-0000-000000000000`, 1),
		strings.Replace(validTrip, `8860b315-ec86-42eb-a17c-7c163d721ff5`, `invalid`, 1),
		validTrip + `{}`,
		`null`,
		`{`,
	} {
		service := &fakeTrips{}
		r := httptest.NewRequest(http.MethodPost, "/api/v1/trips", strings.NewReader(body))
		r.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		Router(fakeDB{}, service, time.Second).ServeHTTP(w, r)
		if w.Code != 400 || service.calls != 0 || w.Header().Get("Content-Type") != "application/problem+json" {
			t.Fatalf("invalid request: code=%d body=%s calls=%d", w.Code, w.Body, service.calls)
		}
		var problem struct{ Code string }
		if err := json.Unmarshal(w.Body.Bytes(), &problem); err != nil || problem.Code != "invalid_request" {
			t.Fatalf("problem: %s; error: %v", w.Body, err)
		}
	}
}

func TestCreateTripDriverBusy(t *testing.T) {
	service := &fakeTrips{err: trips.ErrDriverBusy}
	r := httptest.NewRequest(http.MethodPost, "/api/v1/trips", strings.NewReader(validTrip))
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	Router(fakeDB{}, service, time.Second).ServeHTTP(w, r)
	var problem struct{ Code string }
	if err := json.Unmarshal(w.Body.Bytes(), &problem); err != nil || w.Code != 409 || problem.Code != "driver_busy" || service.calls != 1 {
		t.Fatalf("driver busy: code=%d body=%s error=%v", w.Code, w.Body, err)
	}
}

func TestCreateTripZeroPrice(t *testing.T) {
	service := &fakeTrips{}
	body := strings.Replace(validTrip, `"price":1450`, `"price":0`, 1)
	r := httptest.NewRequest(http.MethodPost, "/api/v1/trips", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	Router(fakeDB{}, service, time.Second).ServeHTTP(w, r)
	if w.Code != 201 || service.calls != 1 {
		t.Fatalf("zero price: code=%d body=%s", w.Code, w.Body)
	}
}

func TestTripErrors(t *testing.T) {
	id := "1f0a9c62-4a1c-4f2e-9d33-2a4bb0f0b111"
	for _, tc := range []struct {
		method string
		path   string
		err    error
		status int
		code   string
	}{
		{http.MethodGet, "/api/v1/trips/invalid", nil, 400, "invalid_request"},
		{http.MethodGet, "/api/v1/trips/" + id, trips.ErrNotFound, 404, "trip_not_found"},
		{http.MethodPost, "/api/v1/trips/" + id + "/finish", trips.ErrCompleted, 409, "trip_completed"},
		{http.MethodPost, "/api/v1/trips/" + id + "/finish", errors.New("database offline"), 500, "internal_error"},
	} {
		service := &fakeTrips{err: tc.err}
		w := httptest.NewRecorder()
		Router(fakeDB{}, service, time.Second).ServeHTTP(w, httptest.NewRequest(tc.method, tc.path, nil))
		var problem struct{ Code string }
		if err := json.Unmarshal(w.Body.Bytes(), &problem); err != nil || w.Code != tc.status || problem.Code != tc.code || w.Header().Get("Content-Type") != "application/problem+json" {
			t.Fatalf("%s: code=%d body=%s; error: %v", tc.path, w.Code, w.Body, err)
		}
	}
}

func TestGetAndFinishTrip(t *testing.T) {
	id := uuid.MustParse("1f0a9c62-4a1c-4f2e-9d33-2a4bb0f0b111")
	started := time.Date(2026, 9, 28, 12, 0, 0, 0, time.UTC)
	finished := started.Add(18 * time.Minute)
	service := &fakeTrips{trip: trips.Trip{ID: id, Status: "completed", StartedAt: started, FinishedAt: &finished}}
	for _, tc := range []struct{ method, path string }{
		{http.MethodGet, "/api/v1/trips/" + id.String()},
		{http.MethodPost, "/api/v1/trips/" + id.String() + "/finish"},
	} {
		w := httptest.NewRecorder()
		Router(fakeDB{}, service, time.Second).ServeHTTP(w, httptest.NewRequest(tc.method, tc.path, nil))
		var body struct {
			Status     string     `json:"status"`
			FinishedAt *time.Time `json:"finished_at"`
		}
		if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil || w.Code != 200 || body.Status != "completed" || body.FinishedAt == nil || !body.FinishedAt.Equal(finished) {
			t.Fatalf("%s %s: code=%d body=%s; error=%v", tc.method, tc.path, w.Code, w.Body, err)
		}
	}
}
