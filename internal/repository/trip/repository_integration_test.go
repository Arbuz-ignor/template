//go:build integration

package triprepo_test

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Arbuz-ignor/template/internal/domain/trip"
	"github.com/Arbuz-ignor/template/internal/infra/postgres"
	triprepo "github.com/Arbuz-ignor/template/internal/repository/trip"
	"github.com/Arbuz-ignor/template/internal/usecase/trip/create"
	"github.com/Arbuz-ignor/template/internal/usecase/trip/finish"
	"github.com/Arbuz-ignor/template/internal/usecase/trip/get"
)

type store struct {
	pool   *pgxpool.Pool
	repo   *triprepo.Repository
	tx     *postgres.TxManager
	create *create.Usecase
	get    *get.Usecase
	finish *finish.Usecase
}
type events struct{}

func (events) Created(context.Context)   {}
func (events) Completed(context.Context) {}
func newStore(t *testing.T) *store {
	t.Helper()
	url := os.Getenv("DATABASE_URL")
	if url == "" {
		t.Fatal("DATABASE_URL is required for integration tests")
	}
	pool, err := pgxpool.New(context.Background(), url)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	repo := triprepo.NewRepository(pool, 5*time.Second)
	tx := postgres.NewTxManager(pool, 5*time.Second)
	logger := slog.New(slog.DiscardHandler)
	return &store{pool: pool, repo: repo, tx: tx, create: create.New(repo, tx, logger, events{}), get: get.New(repo), finish: finish.New(repo, tx, logger, events{})}
}

func (s *store) input(t *testing.T) trip.Trip {
	t.Helper()
	input := trip.Trip{ID: uuid.New(), UserID: uuid.New(), DriverID: uuid.New(), Status: trip.Active, StartedAt: time.Now().UTC(), Price: 100}
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if _, err := s.pool.Exec(ctx, "DELETE FROM trips WHERE driver_id=$1", input.DriverID); err != nil {
			t.Error(err)
		}
	})
	return input
}

func TestLifecycle(t *testing.T) {
	s := newStore(t)
	ctx := context.Background()
	input := s.input(t)
	first, err := s.create.Execute(ctx, input)
	if err != nil {
		t.Fatal(err)
	}
	got, err := s.get.Execute(ctx, first.ID)
	if err != nil || got.Status != trip.Active {
		t.Fatalf("get: %+v %v", got, err)
	}
	if _, err := s.create.Execute(ctx, input); !errors.Is(err, trip.ErrDriverBusy) {
		t.Fatalf("busy: %v", err)
	}
	completed, err := s.finish.Execute(ctx, first.ID)
	if err != nil || completed.Status != trip.Completed || completed.FinishedAt == nil {
		t.Fatalf("finish: %+v %v", completed, err)
	}
	if _, err := s.finish.Execute(ctx, first.ID); !errors.Is(err, trip.ErrCompleted) {
		t.Fatalf("repeat: %v", err)
	}
	got, err = s.get.Execute(ctx, first.ID)
	if err != nil || !got.FinishedAt.Equal(*completed.FinishedAt) {
		t.Fatalf("finish time changed: %+v %v", got, err)
	}
	for _, fetch := range []func(context.Context, uuid.UUID) (trip.Trip, error){s.get.Execute, s.finish.Execute} {
		if _, err := fetch(ctx, uuid.New()); !errors.Is(err, trip.ErrNotFound) {
			t.Fatalf("missing: %v", err)
		}
	}
	if _, err := s.create.Execute(ctx, input); err != nil {
		t.Fatalf("driver remains busy: %v", err)
	}
	var count int
	if err := s.pool.QueryRow(ctx, "SELECT count(*) FROM trip_status_history WHERE trip_id=$1", first.ID).Scan(&count); err != nil || count != 2 {
		t.Fatalf("history=%d error=%v", count, err)
	}
}

func parallel(t *testing.T, count int, call func() error, wantConflict error) {
	t.Helper()
	results := make(chan error, count)
	var wg sync.WaitGroup
	for range count {
		wg.Go(func() { results <- call() })
	}
	wg.Wait()
	close(results)
	success, conflicts := 0, 0
	for err := range results {
		switch {
		case err == nil:
			success++
		case errors.Is(err, wantConflict):
			conflicts++
		default:
			t.Fatal(err)
		}
	}
	if success != 1 || conflicts != count-1 {
		t.Fatalf("success=%d conflicts=%d", success, conflicts)
	}
}

func TestConcurrentCreate(t *testing.T) {
	s := newStore(t)
	input := s.input(t)
	parallel(t, 20, func() error { _, err := s.create.Execute(context.Background(), input); return err }, trip.ErrDriverBusy)
}

func TestConcurrentFinish(t *testing.T) {
	s := newStore(t)
	ctx := context.Background()
	created, err := s.create.Execute(ctx, s.input(t))
	if err != nil {
		t.Fatal(err)
	}
	parallel(t, 2, func() error { _, err := s.finish.Execute(ctx, created.ID); return err }, trip.ErrCompleted)
	var count int
	if err := s.pool.QueryRow(ctx, "SELECT count(*) FROM trip_status_history WHERE trip_id=$1", created.ID).Scan(&count); err != nil || count != 2 {
		t.Fatalf("history=%d error=%v", count, err)
	}
}

func TestBothTablesRollback(t *testing.T) {
	s := newStore(t)
	input := s.input(t)
	ctx := context.Background()
	failure := errors.New("force rollback")
	err := s.tx.Do(ctx, func(ctx context.Context) error {
		if err := s.repo.Create(ctx, input); err != nil {
			return err
		}
		if err := s.repo.AddStatus(ctx, input.ID, nil, trip.Active, "created"); err != nil {
			return err
		}
		return failure
	})
	if !errors.Is(err, failure) {
		t.Fatal(err)
	}
	for _, query := range []string{"SELECT count(*) FROM trips WHERE id=$1", "SELECT count(*) FROM trip_status_history WHERE trip_id=$1"} {
		var count int
		if err := s.pool.QueryRow(ctx, query, input.ID).Scan(&count); err != nil || count != 0 {
			t.Fatalf("rollback count=%d error=%v", count, err)
		}
	}
}

func TestNestedTransactionAndPanic(t *testing.T) {
	s := newStore(t)
	ctx := context.Background()
	input := s.input(t)
	var recovered any
	func() {
		defer func() { recovered = recover() }()
		_ = s.tx.Do(ctx, func(ctx context.Context) error {
			executor := postgres.FromContext(ctx, s.pool)
			return s.tx.Do(ctx, func(ctx context.Context) error {
				if postgres.FromContext(ctx, s.pool) != executor {
					t.Error("nested transaction differs")
				}
				if err := s.repo.Create(ctx, input); err != nil {
					return err
				}
				panic("rollback")
			})
		})
	}()
	if recovered == nil {
		t.Fatal("panic was not propagated")
	}
	if _, err := s.repo.Get(ctx, input.ID); !errors.Is(err, trip.ErrNotFound) {
		t.Fatalf("rollback: %v", err)
	}
	if err := s.repo.Create(ctx, input); err != nil {
		t.Fatalf("outside transaction: %v", err)
	}
	if _, err := s.repo.CountActive(ctx); err != nil {
		t.Fatal(err)
	}
}
