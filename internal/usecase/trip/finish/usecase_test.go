package finish_test

import (
	"context"
	"errors"
	"log/slog"
	"testing"

	"github.com/google/uuid"
	"go.uber.org/mock/gomock"

	"github.com/Arbuz-ignor/template/internal/domain/trip"
	"github.com/Arbuz-ignor/template/internal/usecase/trip/finish"
)

func TestExecute(t *testing.T) {
	failure := errors.New("repository offline")
	for _, tc := range []struct {
		name, status                           string
		fetchErr, saveErr, historyErr, wantErr error
	}{
		{name: "success", status: trip.Active},
		{name: "completed", status: trip.Completed, wantErr: trip.ErrCompleted},
		{name: "missing", fetchErr: trip.ErrNotFound, wantErr: trip.ErrNotFound},
		{name: "fetch failure", fetchErr: failure, wantErr: failure},
		{name: "save failure", status: trip.Active, saveErr: failure, wantErr: failure},
		{name: "history failure", status: trip.Active, historyErr: failure, wantErr: failure},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			repo := NewMocktripRepository(ctrl)
			tx := NewMocktxManager(ctrl)
			id := uuid.New()
			tx.EXPECT().Do(gomock.Any(), gomock.Any()).DoAndReturn(func(ctx context.Context, fn func(context.Context) error) error { return fn(ctx) })
			repo.EXPECT().GetForUpdate(gomock.Any(), id).Return(trip.Trip{ID: id, Status: tc.status}, tc.fetchErr)
			if tc.fetchErr == nil && tc.status == trip.Active {
				repo.EXPECT().SaveCompleted(gomock.Any(), gomock.Any()).Return(tc.saveErr)
				if tc.saveErr == nil {
					repo.EXPECT().AddStatus(gomock.Any(), id, gomock.Any(), trip.Completed, "trip finished").Return(tc.historyErr)
				}
			}
			got, err := finish.New(repo, tx, testLogger(), testEvents(ctrl)).Execute(context.Background(), id)
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("error: %v", err)
			}
			if err == nil && (got.ID != id || got.Status != trip.Completed || got.FinishedAt == nil) {
				t.Fatalf("trip: %+v", got)
			}
		})
	}
}

func testLogger() *slog.Logger { return slog.New(slog.DiscardHandler) }

func testEvents(ctrl *gomock.Controller) *MockeventRecorder {
	events := NewMockeventRecorder(ctrl)
	events.EXPECT().Completed(gomock.Any()).AnyTimes()
	return events
}
