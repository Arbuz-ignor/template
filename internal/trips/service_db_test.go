package trips

import (
	"context"
	"errors"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/Arbuz-ignor/template/internal/database"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestDatabaseLifecycle(t *testing.T) {
	url := os.Getenv("DATABASE_URL")
	if url == "" {
		t.Skip("DATABASE_URL is not set")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	pool, err := pgxpool.New(ctx, url)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	if err := pool.Ping(ctx); err != nil {
		t.Fatal(err)
	}
	repo := NewRepository(pool, 5*time.Second)
	tx := database.NewTxManager(pool, 5*time.Second)
	service := NewService(repo, tx)
	input := Trip{
		UserID: uuid.New(), DriverID: uuid.New(),
		StartPoint: Coordinates{Latitude: 59.9398, Longitude: 30.3146},
		EndPoint:   Coordinates{Latitude: 59.929, Longitude: 30.3626}, Price: 1450,
	}
	cleanup := func(id uuid.UUID) {
		t.Helper()
		cleanupCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if _, err := pool.Exec(cleanupCtx, "DELETE FROM trips WHERE id = $1", id); err != nil {
			t.Error(err)
		}
	}

	first, err := service.Create(ctx, input)
	if err != nil {
		t.Fatal(err)
	}
	defer cleanup(first.ID)
	got, err := service.Get(ctx, first.ID)
	if err != nil || got.ID != first.ID || got.Status != "active" {
		t.Fatalf("get: %+v, %v", got, err)
	}
	if _, err := service.Create(ctx, input); !errors.Is(err, ErrDriverBusy) {
		t.Fatalf("same driver: %v", err)
	}
	finished, err := service.Finish(ctx, first.ID)
	if err != nil || finished.Status != "completed" || finished.FinishedAt == nil {
		t.Fatalf("finish: %+v, %v", finished, err)
	}
	if _, err := service.Finish(ctx, first.ID); !errors.Is(err, ErrCompleted) {
		t.Fatalf("second finish: %v", err)
	}
	got, err = service.Get(ctx, first.ID)
	if err != nil || got.FinishedAt == nil || !got.FinishedAt.Equal(*finished.FinishedAt) {
		t.Fatalf("finish time changed: %+v, %v", got, err)
	}
	missing := uuid.New()
	if _, err := service.Get(ctx, missing); !errors.Is(err, ErrNotFound) {
		t.Fatalf("missing trip: %v", err)
	}
	if _, err := service.Finish(ctx, missing); !errors.Is(err, ErrNotFound) {
		t.Fatalf("finish missing trip: %v", err)
	}
	var count int
	if err := pool.QueryRow(ctx, "SELECT count(*) FROM trip_status_history WHERE trip_id = $1", first.ID).Scan(&count); err != nil || count != 2 {
		t.Fatalf("history: %d, %v", count, err)
	}
	again, err := service.Create(ctx, input)
	if err != nil {
		t.Fatalf("driver after finish: %v", err)
	}
	defer cleanup(again.ID)
	var wg sync.WaitGroup
	finishResults := make(chan error, 2)
	for range 2 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := service.Finish(ctx, again.ID)
			finishResults <- err
		}()
	}
	wg.Wait()
	close(finishResults)
	finishedCount, completedCount := 0, 0
	for err := range finishResults {
		switch {
		case err == nil:
			finishedCount++
		case errors.Is(err, ErrCompleted):
			completedCount++
		default:
			t.Fatalf("concurrent finish: %v", err)
		}
	}
	if finishedCount != 1 || completedCount != 1 {
		t.Fatalf("finished=%d completed=%d", finishedCount, completedCount)
	}
	if err := pool.QueryRow(ctx, "SELECT count(*) FROM trip_status_history WHERE trip_id = $1", again.ID).Scan(&count); err != nil || count != 2 {
		t.Fatalf("concurrent finish history: %d, %v", count, err)
	}

	input.DriverID = uuid.New()
	results := make(chan error, 20)
	ids := make(chan uuid.UUID, 20)
	for range 20 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			trip, err := service.Create(ctx, input)
			if err == nil {
				ids <- trip.ID
			}
			results <- err
		}()
	}
	wg.Wait()
	close(results)
	close(ids)
	for id := range ids {
		defer cleanup(id)
	}
	created, busy := 0, 0
	for err := range results {
		switch {
		case err == nil:
			created++
		case errors.Is(err, ErrDriverBusy):
			busy++
		default:
			t.Fatalf("concurrent create: %v", err)
		}
	}
	if created != 1 || busy != 19 {
		t.Fatalf("created=%d busy=%d", created, busy)
	}

	bad := Trip{ID: uuid.New(), UserID: uuid.New(), DriverID: uuid.New(), StartPoint: input.StartPoint, EndPoint: input.EndPoint, Price: 100, Status: "active", StartedAt: time.Now().UTC()}
	defer cleanup(bad.ID)
	err = tx.Do(ctx, func(ctx context.Context) error {
		if err := repo.Create(ctx, bad); err != nil {
			return err
		}
		executor := database.FromContext(ctx, pool)
		return tx.Do(ctx, func(ctx context.Context) error {
			if database.FromContext(ctx, pool) != executor {
				t.Error("nested Do opened another transaction")
			}
			return repo.AddStatus(ctx, bad.ID, nil, "bad_status", "")
		})
	})
	if err == nil {
		t.Fatal("invalid status was accepted")
	}
	if _, err := service.Get(ctx, bad.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("trip remained after rollback: %v", err)
	}
	bad.ID = uuid.New()
	bad.DriverID = uuid.New()
	defer cleanup(bad.ID)
	var recovered any
	func() {
		defer func() { recovered = recover() }()
		_ = tx.Do(ctx, func(ctx context.Context) error {
			if err := repo.Create(ctx, bad); err != nil {
				return err
			}
			panic("rollback")
		})
	}()
	if recovered == nil {
		t.Fatal("panic was not propagated")
	}
	if _, err := service.Get(ctx, bad.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("trip remained after panic: %v", err)
	}
	bad.ID = uuid.New()
	bad.DriverID = uuid.New()
	defer cleanup(bad.ID)
	if err := repo.Create(ctx, bad); err != nil {
		t.Fatalf("repository outside Do: %v", err)
	}
	if got, err := repo.Get(ctx, bad.ID); err != nil || got.ID != bad.ID {
		t.Fatalf("read outside Do: %+v, %v", got, err)
	}
}
