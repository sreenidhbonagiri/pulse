package repository

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/sreenidhbonagiri/pulse/backend/internal/models"
)

const monitorColumns = `id, user_id, name, url, http_method, check_interval_seconds, timeout_seconds, expected_status_code, is_active, next_check_at, created_at, updated_at`

type PostgresMonitorRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresMonitorRepository(pool *pgxpool.Pool) *PostgresMonitorRepository {
	return &PostgresMonitorRepository{pool: pool}
}

func (r *PostgresMonitorRepository) Create(ctx context.Context, monitor *models.Monitor) error {
	if monitor.ID == uuid.Nil {
		monitor.ID = uuid.New()
	}
	if monitor.NextCheckAt == nil {
		now := time.Now().UTC()
		monitor.NextCheckAt = &now
	}

	row := r.pool.QueryRow(ctx, `
		INSERT INTO monitors (
			id, user_id, name, url, http_method,
			check_interval_seconds, timeout_seconds, expected_status_code, is_active, next_check_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING `+monitorColumns,
		monitor.ID,
		monitor.UserID,
		monitor.Name,
		monitor.URL,
		monitor.HTTPMethod,
		monitor.CheckIntervalSeconds,
		monitor.TimeoutSeconds,
		monitor.ExpectedStatusCode,
		monitor.IsActive,
		monitor.NextCheckAt,
	)

	scanned, err := scanMonitor(row)
	if err != nil {
		return err
	}
	*monitor = *scanned
	return nil
}

func (r *PostgresMonitorRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.Monitor, error) {
	monitor, err := scanMonitor(r.pool.QueryRow(ctx, `
		SELECT `+monitorColumns+`
		FROM monitors
		WHERE id = $1
	`, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return monitor, nil
}

func (r *PostgresMonitorRepository) List(ctx context.Context) ([]models.Monitor, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT `+monitorColumns+`
		FROM monitors
		ORDER BY created_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return collectMonitors(rows)
}

func (r *PostgresMonitorRepository) ListDue(ctx context.Context, now time.Time, limit int) ([]models.Monitor, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT `+monitorColumns+`
		FROM monitors
		WHERE is_active = TRUE
		  AND next_check_at IS NOT NULL
		  AND next_check_at <= $1
		ORDER BY next_check_at ASC
		LIMIT $2
	`, now, clampDueLimit(limit))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return collectMonitors(rows)
}

func (r *PostgresMonitorRepository) ClaimDue(ctx context.Context, now time.Time, enqueue func(models.Monitor) error) (*models.Monitor, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	monitor, err := scanMonitor(tx.QueryRow(ctx, `
		SELECT `+monitorColumns+`
		FROM monitors
		WHERE is_active = TRUE
		  AND next_check_at IS NOT NULL
		  AND next_check_at <= $1
		ORDER BY next_check_at ASC
		LIMIT 1
		FOR UPDATE SKIP LOCKED
	`, now))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	if err := enqueue(*monitor); err != nil {
		return monitor, err
	}

	next := NextCheckTime(time.Now().UTC(), monitor.CheckIntervalSeconds)
	if _, err := tx.Exec(ctx, `
		UPDATE monitors
		SET next_check_at = $2, updated_at = NOW()
		WHERE id = $1
	`, monitor.ID, next); err != nil {
		return monitor, err
	}

	if err := tx.Commit(ctx); err != nil {
		return monitor, err
	}

	monitor.NextCheckAt = &next
	return monitor, nil
}

func (r *PostgresMonitorRepository) Update(ctx context.Context, monitor *models.Monitor) error {
	row := r.pool.QueryRow(ctx, `
		UPDATE monitors
		SET
			user_id = $2,
			name = $3,
			url = $4,
			http_method = $5,
			check_interval_seconds = $6,
			timeout_seconds = $7,
			expected_status_code = $8,
			is_active = $9,
			next_check_at = $10,
			updated_at = NOW()
		WHERE id = $1
		RETURNING `+monitorColumns,
		monitor.ID,
		monitor.UserID,
		monitor.Name,
		monitor.URL,
		monitor.HTTPMethod,
		monitor.CheckIntervalSeconds,
		monitor.TimeoutSeconds,
		monitor.ExpectedStatusCode,
		monitor.IsActive,
		monitor.NextCheckAt,
	)

	scanned, err := scanMonitor(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	if err != nil {
		return err
	}
	*monitor = *scanned
	return nil
}

func (r *PostgresMonitorRepository) Delete(ctx context.Context, id uuid.UUID) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM monitors WHERE id = $1`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

type scanner interface {
	Scan(dest ...any) error
}

func collectMonitors(rows pgx.Rows) ([]models.Monitor, error) {
	monitors := make([]models.Monitor, 0)
	for rows.Next() {
		monitor, err := scanMonitor(rows)
		if err != nil {
			return nil, err
		}
		monitors = append(monitors, *monitor)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return monitors, nil
}

func scanMonitor(row scanner) (*models.Monitor, error) {
	var monitor models.Monitor
	err := row.Scan(
		&monitor.ID,
		&monitor.UserID,
		&monitor.Name,
		&monitor.URL,
		&monitor.HTTPMethod,
		&monitor.CheckIntervalSeconds,
		&monitor.TimeoutSeconds,
		&monitor.ExpectedStatusCode,
		&monitor.IsActive,
		&monitor.NextCheckAt,
		&monitor.CreatedAt,
		&monitor.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &monitor, nil
}
