package repository

import (
	"context"
	"errors"
	"sync"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type txContextKey struct{}

func withTx(ctx context.Context, tx pgx.Tx) context.Context {
	return context.WithValue(ctx, txContextKey{}, tx)
}

func txFromContext(ctx context.Context) pgx.Tx {
	tx, _ := ctx.Value(txContextKey{}).(pgx.Tx)
	return tx
}

type querier interface {
	Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, arguments ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, arguments ...any) pgx.Row
}

func querierFrom(ctx context.Context, pool *pgxpool.Pool) querier {
	if tx := txFromContext(ctx); tx != nil {
		return tx
	}
	return pool
}

// Transactor runs work while a monitor row is locked so two workers cannot
// open two incidents for the same outage.
type Transactor interface {
	WithMonitorLock(ctx context.Context, monitorID uuid.UUID, fn func(ctx context.Context) error) error
}

type PostgresTransactor struct {
	pool *pgxpool.Pool
}

func NewPostgresTransactor(pool *pgxpool.Pool) *PostgresTransactor {
	return &PostgresTransactor{pool: pool}
}

func (t *PostgresTransactor) WithMonitorLock(ctx context.Context, monitorID uuid.UUID, fn func(ctx context.Context) error) error {
	tx, err := t.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	var id uuid.UUID
	err = tx.QueryRow(ctx, `
		SELECT id
		FROM monitors
		WHERE id = $1
		FOR UPDATE
	`, monitorID).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	if err != nil {
		return err
	}

	if err := fn(withTx(ctx, tx)); err != nil {
		return err
	}

	return tx.Commit(ctx)
}

// MemoryTransactor serializes incident evaluation in tests.
type MemoryTransactor struct {
	mu sync.Mutex
}

func NewMemoryTransactor() *MemoryTransactor {
	return &MemoryTransactor{}
}

func (t *MemoryTransactor) WithMonitorLock(ctx context.Context, _ uuid.UUID, fn func(ctx context.Context) error) error {
	t.mu.Lock()
	defer t.mu.Unlock()
	return fn(ctx)
}
