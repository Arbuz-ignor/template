package triprepo

import (
	"context"
	"errors"
	"fmt"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	domain "github.com/Arbuz-ignor/template/internal/domain/trip"
	"github.com/Arbuz-ignor/template/internal/infra/postgres"
)

var sql = sq.StatementBuilder.PlaceholderFormat(sq.Dollar)

const columns = "id, user_id, driver_id, start_latitude, start_longitude, end_latitude, end_longitude, price, status, started_at, finished_at"

type Repository struct {
	pool    *pgxpool.Pool
	timeout time.Duration
}

func NewRepository(pool *pgxpool.Pool, timeout time.Duration) *Repository {
	return &Repository{pool: pool, timeout: timeout}
}

func (r *Repository) Create(ctx context.Context, trip domain.Trip) error {
	query, args, err := sql.Insert("trips").Columns(
		"id", "user_id", "driver_id", "start_latitude", "start_longitude", "end_latitude", "end_longitude", "price", "status", "started_at",
	).Values(
		trip.ID, trip.UserID, trip.DriverID, trip.StartPoint.Latitude, trip.StartPoint.Longitude,
		trip.EndPoint.Latitude, trip.EndPoint.Longitude, trip.Price, trip.Status, trip.StartedAt,
	).ToSql()
	if err != nil {
		return fmt.Errorf("build trip insert: %w", err)
	}
	queryCtx, cancel := context.WithTimeout(ctx, r.timeout)
	defer cancel()
	_, err = postgres.FromContext(ctx, r.pool).Exec(queryCtx, query, args...)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" && pgErr.ConstraintName == "trips_one_active_per_driver_idx" {
			return domain.ErrDriverBusy
		}
		return fmt.Errorf("insert trip: %w", err)
	}
	return nil
}

func (r *Repository) AddStatus(ctx context.Context, tripID uuid.UUID, from *string, to, reason string) error {
	query, args, err := sql.Insert("trip_status_history").Columns("trip_id", "from_status", "to_status", "reason").Values(tripID, from, to, reason).ToSql()
	if err != nil {
		return fmt.Errorf("build status insert: %w", err)
	}
	queryCtx, cancel := context.WithTimeout(ctx, r.timeout)
	defer cancel()
	if _, err := postgres.FromContext(ctx, r.pool).Exec(queryCtx, query, args...); err != nil {
		return fmt.Errorf("insert trip status: %w", err)
	}
	return nil
}

func (r *Repository) Get(ctx context.Context, id uuid.UUID) (domain.Trip, error) {
	query, args, err := sql.Select(columns).From("trips").Where(sq.Eq{"id": id}).ToSql()
	if err != nil {
		return domain.Trip{}, fmt.Errorf("build trip select: %w", err)
	}
	queryCtx, cancel := context.WithTimeout(ctx, r.timeout)
	defer cancel()
	trip, err := scanTrip(postgres.FromContext(ctx, r.pool).QueryRow(queryCtx, query, args...))
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Trip{}, domain.ErrNotFound
	}
	if err != nil {
		return domain.Trip{}, fmt.Errorf("select trip: %w", err)
	}
	return trip, nil
}

func (r *Repository) GetForUpdate(ctx context.Context, id uuid.UUID) (domain.Trip, error) {
	query, args, err := sql.Select(columns).From("trips").Where(sq.Eq{"id": id}).Suffix("FOR UPDATE").ToSql()
	if err != nil {
		return domain.Trip{}, fmt.Errorf("build locked trip select: %w", err)
	}
	queryCtx, cancel := context.WithTimeout(ctx, r.timeout)
	defer cancel()
	result, err := scanTrip(postgres.FromContext(ctx, r.pool).QueryRow(queryCtx, query, args...))
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Trip{}, domain.ErrNotFound
	}
	if err != nil {
		return domain.Trip{}, fmt.Errorf("lock trip: %w", err)
	}
	return result, nil
}

func (r *Repository) SaveCompleted(ctx context.Context, result domain.Trip) error {
	query, args, err := sql.Update("trips").Set("status", result.Status).Set("finished_at", result.FinishedAt).Set("updated_at", result.FinishedAt).Where(sq.Eq{"id": result.ID, "status": domain.Active}).ToSql()
	if err != nil {
		return fmt.Errorf("build completed trip update: %w", err)
	}
	queryCtx, cancel := context.WithTimeout(ctx, r.timeout)
	defer cancel()
	tag, err := postgres.FromContext(ctx, r.pool).Exec(queryCtx, query, args...)
	if err != nil {
		return fmt.Errorf("save completed trip: %w", err)
	}
	if tag.RowsAffected() != 1 {
		return domain.ErrCompleted
	}
	return nil
}

func scanTrip(row pgx.Row) (domain.Trip, error) {
	var result domain.Trip
	err := row.Scan(&result.ID, &result.UserID, &result.DriverID,
		&result.StartPoint.Latitude, &result.StartPoint.Longitude,
		&result.EndPoint.Latitude, &result.EndPoint.Longitude,
		&result.Price, &result.Status, &result.StartedAt, &result.FinishedAt)
	return result, err
}

func (r *Repository) CountActive(ctx context.Context) (int64, error) {
	queryCtx, cancel := context.WithTimeout(ctx, r.timeout)
	defer cancel()
	var count int64
	if err := r.pool.QueryRow(queryCtx, "SELECT count(*) FROM trips WHERE status = $1", domain.Active).Scan(&count); err != nil {
		return 0, fmt.Errorf("count active trips: %w", err)
	}
	return count, nil
}
