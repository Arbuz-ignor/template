package get_test

import (
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"go.uber.org/mock/gomock"

	"github.com/Arbuz-ignor/template/internal/domain/trip"
	"github.com/Arbuz-ignor/template/internal/http/trip/get"
)

func TestHandle(t *testing.T) {
	for _, wantErr := range []error{nil, trip.ErrNotFound} {
		ctrl := gomock.NewController(t)
		u := NewMockusecase(ctrl)
		id := uuid.New()
		u.EXPECT().Execute(gomock.Any(), id).Return(trip.Trip{ID: id}, wantErr)
		w := httptest.NewRecorder()
		get.New(u).Handle(w, httptest.NewRequest("GET", "/api/v1/trips/"+id.String(), nil), id)
		want := 200
		if wantErr != nil {
			want = 404
		}
		if w.Code != want {
			t.Fatalf("response: %d %s", w.Code, w.Body)
		}
	}
}
