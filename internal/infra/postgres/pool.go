package postgres

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Arbuz-ignor/template/internal/config"
)

func Open(ctx context.Context, c config.Config, tracer pgx.QueryTracer) (*pgxpool.Pool, error) {
	p, err := pgxpool.ParseConfig(c.DatabaseURL)
	if err != nil {
		return nil, fmt.Errorf("parse DATABASE_URL: %w", err)
	}
	p.ConnConfig.Tracer = tracer
	p.MaxConns = c.DatabaseMaxConns
	p.MinConns = c.DatabaseMinConns
	p.MaxConnLifetime = c.DatabaseMaxLifetime
	p.ConnConfig.ConnectTimeout = c.DatabaseConnectTime

	pool, err := pgxpool.NewWithConfig(ctx, p)
	if err != nil {
		return nil, fmt.Errorf("create database pool: %w", err)
	}
	pingCtx, cancel := context.WithTimeout(ctx, c.DatabaseConnectTime)
	defer cancel()
	if err := pool.Ping(pingCtx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping database: %w", err)
	}
	return pool, nil
}
