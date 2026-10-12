package database

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type txKey struct{}

type Executor interface {
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
	QueryRow(context.Context, string, ...any) pgx.Row
}

type TxManager struct {
	pool    *pgxpool.Pool
	timeout time.Duration
}

func NewTxManager(pool *pgxpool.Pool, timeout time.Duration) *TxManager {
	return &TxManager{pool: pool, timeout: timeout}
}

func (m *TxManager) Do(ctx context.Context, fn func(context.Context) error) (err error) {
	if _, ok := ctx.Value(txKey{}).(pgx.Tx); ok {
		return fn(ctx)
	}

	beginCtx, cancelBegin := context.WithTimeout(ctx, m.timeout)
	defer cancelBegin()
	tx, err := m.pool.BeginTx(beginCtx, pgx.TxOptions{IsoLevel: pgx.ReadCommitted})
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer func() {
		rollbackCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), m.timeout)
		defer cancel()
		if rollbackErr := tx.Rollback(rollbackCtx); rollbackErr != nil && !errors.Is(rollbackErr, pgx.ErrTxClosed) && err != nil {
			err = errors.Join(err, fmt.Errorf("rollback transaction: %w", rollbackErr))
		}
	}()

	if err := fn(context.WithValue(ctx, txKey{}, tx)); err != nil {
		return err
	}
	commitCtx, cancelCommit := context.WithTimeout(ctx, m.timeout)
	defer cancelCommit()
	if err := tx.Commit(commitCtx); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}
	return nil
}

func FromContext(ctx context.Context, pool *pgxpool.Pool) Executor {
	if tx, ok := ctx.Value(txKey{}).(pgx.Tx); ok {
		return tx
	}
	return pool
}
