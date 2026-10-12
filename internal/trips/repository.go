package trips

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/Arbuz-ignor/template/internal/database"
	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
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

func (r *Repository) Create(ctx context.Context, trip Trip) error {
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
	_, err = database.FromContext(ctx, r.pool).Exec(queryCtx, query, args...)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" && pgErr.ConstraintName == "trips_one_active_per_driver_idx" {
			return ErrDriverBusy
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
	if _, err := database.FromContext(ctx, r.pool).Exec(queryCtx, query, args...); err != nil {
		return fmt.Errorf("insert trip status: %w", err)
	}
	return nil
}

func (r *Repository) Get(ctx context.Context, id uuid.UUID) (Trip, error) {
	query, args, err := sql.Select(columns).From("trips").Where(sq.Eq{"id": id}).ToSql()
	if err != nil {
		return Trip{}, fmt.Errorf("build trip select: %w", err)
	}
	queryCtx, cancel := context.WithTimeout(ctx, r.timeout)
	defer cancel()
	trip, err := scanTrip(database.FromContext(ctx, r.pool).QueryRow(queryCtx, query, args...))
	if errors.Is(err, pgx.ErrNoRows) {
		return Trip{}, ErrNotFound
	}
	if err != nil {
		return Trip{}, fmt.Errorf("select trip: %w", err)
	}
	return trip, nil
}

func (r *Repository) Finish(ctx context.Context, id uuid.UUID, at time.Time) (Trip, error) {
	query, args, err := sql.Update("trips").Set("status", "completed").Set("finished_at", at).Set("updated_at", at).
		Where(sq.Eq{"id": id, "status": "active"}).Suffix("RETURNING " + columns).ToSql()
	if err != nil {
		return Trip{}, fmt.Errorf("build trip update: %w", err)
	}
	queryCtx, cancel := context.WithTimeout(ctx, r.timeout)
	defer cancel()
	trip, err := scanTrip(database.FromContext(ctx, r.pool).QueryRow(queryCtx, query, args...))
	if errors.Is(err, pgx.ErrNoRows) {
		_, getErr := r.Get(ctx, id)
		if getErr == nil {
			return Trip{}, ErrCompleted
		}
		return Trip{}, getErr
	}
	if err != nil {
		return Trip{}, fmt.Errorf("finish trip: %w", err)
	}
	return trip, nil
}

func scanTrip(row pgx.Row) (Trip, error) {
	var trip Trip
	err := row.Scan(&trip.ID, &trip.UserID, &trip.DriverID,
		&trip.StartPoint.Latitude, &trip.StartPoint.Longitude,
		&trip.EndPoint.Latitude, &trip.EndPoint.Longitude,
		&trip.Price, &trip.Status, &trip.StartedAt, &trip.FinishedAt)
	return trip, err
}
