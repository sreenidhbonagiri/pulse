package repository

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/sreenidhbonagiri/pulse/backend/internal/models"
)

const monitorColumns = `id, user_id, name, url, http_method, check_interval_seconds, timeout_seconds, expected_status_code, is_active, created_at, updated_at`

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

	row := r.pool.QueryRow(ctx, `
		INSERT INTO monitors (
			id, user_id, name, url, http_method,
			check_interval_seconds, timeout_seconds, expected_status_code, is_active
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
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
		&monitor.CreatedAt,
		&monitor.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &monitor, nil
}
