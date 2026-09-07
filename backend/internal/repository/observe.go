package repository

import (
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/sreenidhbonagiri/pulse/backend/internal/metrics"
)

func mapUniqueViolation(err error) error {
	if err == nil {
		return nil
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return ErrDuplicate
	}
	return observeDBError(err)
}

func observeDBError(err error) error {
	if err == nil || errors.Is(err, pgx.ErrNoRows) || errors.Is(err, ErrNotFound) || errors.Is(err, ErrDuplicate) {
		return err
	}
	metrics.DatabaseError()
	return err
}

func notFoundOrDB(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	return observeDBError(err)
}
