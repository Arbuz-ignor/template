package create_test

import (
	"context"
	"errors"
	"log/slog"
	"testing"

	"github.com/google/uuid"
	"go.uber.org/mock/gomock"

	"github.com/Arbuz-ignor/template/internal/domain/trip"
	"github.com/Arbuz-ignor/template/internal/usecase/trip/create"
)

func TestExecute(t *testing.T) {
	failure := errors.New("database offline")
	for _, tc := range []struct {
		name               string
		repoErr, errorWant error
	}{{"success", nil, nil}, {"busy", trip.ErrDriverBusy, trip.ErrDriverBusy}, {"repository error", failure, failure}} {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			repo := NewMocktripRepository(ctrl)
			tx := NewMocktxManager(ctrl)
			tx.EXPECT().Do(gomock.Any(), gomock.Any()).DoAndReturn(func(ctx context.Context, fn func(context.Context) error) error { return fn(ctx) })
			repo.EXPECT().Create(gomock.Any(), gomock.Any()).Return(tc.repoErr)
			if tc.repoErr == nil {
				repo.EXPECT().AddStatus(gomock.Any(), gomock.Any(), nil, trip.Active, "trip created").Return(nil)
			}
			got, err := create.New(repo, tx, testLogger(), testEvents(ctrl)).Execute(context.Background(), trip.Trip{UserID: uuid.New(), DriverID: uuid.New()})
			if !errors.Is(err, tc.errorWant) {
				t.Fatalf("error: %v", err)
			}
			if err == nil && (got.ID == uuid.Nil || got.Status != trip.Active || got.StartedAt.IsZero()) {
				t.Fatalf("trip: %+v", got)
			}
		})
	}
}

func TestInvalidInput(t *testing.T) {
	ctrl := gomock.NewController(t)
	_, err := create.New(NewMocktripRepository(ctrl), NewMocktxManager(ctrl), testLogger(), testEvents(ctrl)).Execute(context.Background(), trip.Trip{})
	var validation *trip.ValidationError
	if !errors.As(err, &validation) {
		t.Fatalf("error: %v", err)
	}
}

func TestHistoryFailure(t *testing.T) {
	ctrl := gomock.NewController(t)
	repo := NewMocktripRepository(ctrl)
	tx := NewMocktxManager(ctrl)
	failure := errors.New("history unavailable")
	tx.EXPECT().Do(gomock.Any(), gomock.Any()).DoAndReturn(func(ctx context.Context, fn func(context.Context) error) error { return fn(ctx) })
	repo.EXPECT().Create(gomock.Any(), gomock.Any()).Return(nil)
	repo.EXPECT().AddStatus(gomock.Any(), gomock.Any(), nil, trip.Active, "trip created").Return(failure)
	_, err := create.New(repo, tx, testLogger(), testEvents(ctrl)).Execute(context.Background(), trip.Trip{UserID: uuid.New(), DriverID: uuid.New()})
	if !errors.Is(err, failure) {
		t.Fatal(err)
	}
}

func testLogger() *slog.Logger { return slog.New(slog.DiscardHandler) }

func testEvents(ctrl *gomock.Controller) *MockeventRecorder {
	events := NewMockeventRecorder(ctrl)
	events.EXPECT().Created(gomock.Any()).AnyTimes()
	return events
}
