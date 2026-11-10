package create_test

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
	"go.uber.org/mock/gomock"

	"github.com/Arbuz-ignor/template/internal/domain/trip"
	"github.com/Arbuz-ignor/template/internal/http/trip/create"
)

const validTrip = `{"user_id":"5cb72c04-7650-45c9-a79b-bcdba0631e0c","driver_id":"8860b315-ec86-42eb-a17c-7c163d721ff5","start_point":{"latitude":59.9398,"longitude":30.3146},"end_point":{"latitude":59.929,"longitude":30.3626},"price":1450}`

func TestHandle(t *testing.T) {
	for _, tc := range []struct {
		name   string
		err    error
		status int
	}{{"success", nil, 201}, {"busy", trip.ErrDriverBusy, 409}, {"failure", errors.New("secret sql password"), 500}} {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			u := NewMockusecase(ctrl)
			id := uuid.New()
			u.EXPECT().Execute(gomock.Any(), gomock.Any()).Return(trip.Trip{ID: id, Status: trip.Active}, tc.err)
			r := httptest.NewRequest(http.MethodPost, "/api/v1/trips", strings.NewReader(validTrip))
			r.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			create.New(u).Handle(w, r)
			if w.Code != tc.status || strings.Contains(w.Body.String(), "secret") {
				t.Fatalf("response: %d %s", w.Code, w.Body)
			}
			if tc.err == nil && w.Header().Get("Location") != "/api/v1/trips/"+id.String() {
				t.Fatal(w.Header())
			}
		})
	}
}

func TestInvalidBody(t *testing.T) {
	for _, body := range []string{"{", "null", validTrip + "{}", strings.Replace(validTrip, `"price":1450`, `"price":null`, 1), strings.Replace(validTrip, `"price":1450`, `"price":-1`, 1), strings.Replace(validTrip, `"price":1450`, `"extra":true`, 1), strings.Replace(validTrip, `"latitude":59.9398`, `"latitude":91`, 1), strings.Replace(validTrip, `"latitude":59.9398,`, "", 1), strings.Replace(validTrip, `"latitude":59.9398`, `"latitude":null`, 1), strings.Replace(validTrip, `8860b315-ec86-42eb-a17c-7c163d721ff5`, "invalid", 1)} {
		ctrl := gomock.NewController(t)
		r := httptest.NewRequest(http.MethodPost, "/api/v1/trips", strings.NewReader(body))
		r.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		create.New(NewMockusecase(ctrl)).Handle(w, r)
		var problem struct{ Code string }
		if err := json.Unmarshal(w.Body.Bytes(), &problem); err != nil || w.Code != 400 || problem.Code != "invalid_request" {
			t.Fatalf("response: %d %s", w.Code, w.Body)
		}
	}
}

func TestContentType(t *testing.T) {
	ctrl := gomock.NewController(t)
	w := httptest.NewRecorder()
	create.New(NewMockusecase(ctrl)).Handle(w, httptest.NewRequest(http.MethodPost, "/api/v1/trips", strings.NewReader(validTrip)))
	if w.Code != 400 {
		t.Fatal(w.Code)
	}
}
