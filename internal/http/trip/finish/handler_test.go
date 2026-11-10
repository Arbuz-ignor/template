package finish_test

import (
	"github.com/Arbuz-ignor/template/internal/domain/trip"
	"github.com/Arbuz-ignor/template/internal/http/trip/finish"
	"github.com/google/uuid"
	"go.uber.org/mock/gomock"
	"net/http/httptest"
	"testing"
)

func TestHandle(t *testing.T) {
	for _, wantErr := range []error{nil, trip.ErrCompleted} {
		ctrl := gomock.NewController(t)
		u := NewMockusecase(ctrl)
		id := uuid.New()
		u.EXPECT().Execute(gomock.Any(), id).Return(trip.Trip{ID: id}, wantErr)
		w := httptest.NewRecorder()
		finish.New(u).Handle(w, httptest.NewRequest("GET", "/api/v1/trips/"+id.String(), nil), id)
		want := 200
		if wantErr != nil {
			want = 409
		}
		if w.Code != want {
			t.Fatalf("response: %d %s", w.Code, w.Body)
		}
	}
}
