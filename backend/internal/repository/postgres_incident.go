package repository

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/sreenidhbonagiri/pulse/backend/internal/models"
)

const incidentColumns = `id, monitor_id, started_at, resolved_at, status, failure_count, created_at, updated_at`

type PostgresIncidentRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresIncidentRepository(pool *pgxpool.Pool) *PostgresIncidentRepository {
	return &PostgresIncidentRepository{pool: pool}
}

func (r *PostgresIncidentRepository) q(ctx context.Context) querier {
	return querierFrom(ctx, r.pool)
}

func (r *PostgresIncidentRepository) Create(ctx context.Context, incident *models.Incident) error {
	if incident.ID == uuid.Nil {
		incident.ID = uuid.New()
	}
	if incident.Status == "" {
		incident.Status = models.IncidentStatusOpen
	}
	if incident.StartedAt.IsZero() {
		incident.StartedAt = time.Now().UTC()
	}
	if incident.FailureCount < 1 {
		incident.FailureCount = 1
	}

	row := r.q(ctx).QueryRow(ctx, `
		INSERT INTO incidents (
			id, monitor_id, started_at, resolved_at, status, failure_count
		)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING `+incidentColumns,
		incident.ID,
		incident.MonitorID,
		incident.StartedAt,
		incident.ResolvedAt,
		incident.Status,
		incident.FailureCount,
	)

	scanned, err := scanIncident(row)
	if err != nil {
		return mapUniqueViolation(err)
	}
	*incident = *scanned
	return nil
}

func (r *PostgresIncidentRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.Incident, error) {
	incident, err := scanIncident(r.q(ctx).QueryRow(ctx, `
		SELECT `+incidentColumns+`
		FROM incidents
		WHERE id = $1
	`, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return incident, nil
}

func (r *PostgresIncidentRepository) GetOpenByMonitorID(ctx context.Context, monitorID uuid.UUID) (*models.Incident, error) {
	incident, err := scanIncident(r.q(ctx).QueryRow(ctx, `
		SELECT `+incidentColumns+`
		FROM incidents
		WHERE monitor_id = $1 AND status = $2
	`, monitorID, models.IncidentStatusOpen))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return incident, nil
}

func (r *PostgresIncidentRepository) ListByMonitorID(ctx context.Context, monitorID uuid.UUID, limit int) ([]models.Incident, error) {
	rows, err := r.q(ctx).Query(ctx, `
		SELECT `+incidentColumns+`
		FROM incidents
		WHERE monitor_id = $1
		ORDER BY started_at DESC
		LIMIT $2
	`, monitorID, clampIncidentLimit(limit))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	incidents := make([]models.Incident, 0)
	for rows.Next() {
		incident, err := scanIncident(rows)
		if err != nil {
			return nil, err
		}
		incidents = append(incidents, *incident)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return incidents, nil
}

func (r *PostgresIncidentRepository) IncrementFailureCount(ctx context.Context, id uuid.UUID, failureCount int) (*models.Incident, error) {
	incident, err := scanIncident(r.q(ctx).QueryRow(ctx, `
		UPDATE incidents
		SET failure_count = $2, updated_at = NOW()
		WHERE id = $1 AND status = $3
		RETURNING `+incidentColumns,
		id,
		failureCount,
		models.IncidentStatusOpen,
	))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return incident, nil
}

func (r *PostgresIncidentRepository) Resolve(ctx context.Context, id uuid.UUID, resolvedAt time.Time) (*models.Incident, error) {
	if resolvedAt.IsZero() {
		resolvedAt = time.Now().UTC()
	}

	incident, err := scanIncident(r.q(ctx).QueryRow(ctx, `
		UPDATE incidents
		SET status = $2, resolved_at = $3, updated_at = NOW()
		WHERE id = $1 AND status = $4
		RETURNING `+incidentColumns,
		id,
		models.IncidentStatusResolved,
		resolvedAt,
		models.IncidentStatusOpen,
	))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return incident, nil
}

func (r *PostgresIncidentRepository) CountByMonitorID(ctx context.Context, monitorID uuid.UUID, since time.Time) (int, error) {
	var count int
	err := r.q(ctx).QueryRow(ctx, `
		SELECT COUNT(*)::int
		FROM incidents
		WHERE monitor_id = $1
		  AND started_at >= $2
	`, monitorID, since).Scan(&count)
	if err != nil {
		return 0, err
	}
	return count, nil
}

func scanIncident(row scanner) (*models.Incident, error) {
	var incident models.Incident
	err := row.Scan(
		&incident.ID,
		&incident.MonitorID,
		&incident.StartedAt,
		&incident.ResolvedAt,
		&incident.Status,
		&incident.FailureCount,
		&incident.CreatedAt,
		&incident.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &incident, nil
}

func mapUniqueViolation(err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return ErrDuplicate
	}
	return err
}
