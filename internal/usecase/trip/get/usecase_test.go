package get_test

import (
	"context"
	"errors"
	"github.com/Arbuz-ignor/template/internal/domain/trip"
	"github.com/Arbuz-ignor/template/internal/usecase/trip/get"
	"github.com/google/uuid"
	"go.uber.org/mock/gomock"
	"testing"
)

func TestExecute(t *testing.T) {
	for _, wantErr := range []error{nil, trip.ErrNotFound, errors.New("database offline")} {
		ctrl := gomock.NewController(t)
		repo := NewMocktripRepository(ctrl)
		id := uuid.New()
		repo.EXPECT().Get(gomock.Any(), id).Return(trip.Trip{ID: id}, wantErr)
		got, err := get.New(repo).Execute(context.Background(), id)
		if !errors.Is(err, wantErr) {
			t.Fatalf("error: %v", err)
		}
		if err == nil && got.ID != id {
			t.Fatalf("trip: %+v", got)
		}
	}
}
