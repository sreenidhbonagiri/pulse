package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/sreenidhbonagiri/pulse/backend/internal/models"
)

const checkResultColumns = `id, job_id, monitor_id, status_code, response_time_ms, success, error_message, checked_at`

type PostgresCheckResultRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresCheckResultRepository(pool *pgxpool.Pool) *PostgresCheckResultRepository {
	return &PostgresCheckResultRepository{pool: pool}
}

func (r *PostgresCheckResultRepository) q(ctx context.Context) querier {
	return querierFrom(ctx, r.pool)
}

func (r *PostgresCheckResultRepository) Create(ctx context.Context, result *models.CheckResult) error {
	if result.ID == uuid.Nil {
		result.ID = uuid.New()
	}
	if result.CheckedAt.IsZero() {
		result.CheckedAt = time.Now().UTC()
	}

	row := r.q(ctx).QueryRow(ctx, `
		INSERT INTO check_results (
			id, job_id, monitor_id, status_code, response_time_ms, success, error_message, checked_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING `+checkResultColumns,
		result.ID,
		nullableUUID(result.JobID),
		result.MonitorID,
		result.StatusCode,
		result.ResponseTimeMs,
		result.Success,
		result.ErrorMessage,
		result.CheckedAt,
	)

	scanned, err := scanCheckResult(row)
	if err != nil {
		return mapUniqueViolation(err)
	}
	*result = *scanned
	return nil
}

func (r *PostgresCheckResultRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.CheckResult, error) {
	result, err := scanCheckResult(r.q(ctx).QueryRow(ctx, `
		SELECT `+checkResultColumns+`
		FROM check_results
		WHERE id = $1
	`, id))
	if err != nil {
		return nil, notFoundOrDB(err)
	}
	return result, nil
}

func (r *PostgresCheckResultRepository) GetByJobID(ctx context.Context, jobID uuid.UUID) (*models.CheckResult, error) {
	result, err := scanCheckResult(r.q(ctx).QueryRow(ctx, `
		SELECT `+checkResultColumns+`
		FROM check_results
		WHERE job_id = $1
	`, jobID))
	if err != nil {
		return nil, notFoundOrDB(err)
	}
	return result, nil
}

func (r *PostgresCheckResultRepository) ListByMonitorID(ctx context.Context, monitorID uuid.UUID, limit int) ([]models.CheckResult, error) {
	rows, err := r.q(ctx).Query(ctx, `
		SELECT `+checkResultColumns+`
		FROM check_results
		WHERE monitor_id = $1
		ORDER BY checked_at DESC
		LIMIT $2
	`, monitorID, clampCheckResultLimit(limit))
	if err != nil {
		return nil, observeDBError(err)
	}
	defer rows.Close()

	results := make([]models.CheckResult, 0)
	for rows.Next() {
		result, err := scanCheckResult(rows)
		if err != nil {
			return nil, observeDBError(err)
		}
		results = append(results, *result)
	}
	if err := rows.Err(); err != nil {
		return nil, observeDBError(err)
	}

	return results, nil
}

func (r *PostgresCheckResultRepository) GetCheckStats(ctx context.Context, monitorID uuid.UUID, since time.Time) (CheckStats, error) {
	var stats CheckStats
	var avg, p50, p95, p99 *float64
	err := r.q(ctx).QueryRow(ctx, `
		SELECT
			(SELECT success
			 FROM check_results
			 WHERE monitor_id = $1
			 ORDER BY checked_at DESC
			 LIMIT 1) AS latest_success,
			COUNT(*)::int AS total_checks,
			COUNT(*) FILTER (WHERE success = FALSE)::int AS failed_checks,
			AVG(response_time_ms) AS avg_latency,
			percentile_cont(0.50) WITHIN GROUP (ORDER BY response_time_ms) AS p50,
			percentile_cont(0.95) WITHIN GROUP (ORDER BY response_time_ms) AS p95,
			percentile_cont(0.99) WITHIN GROUP (ORDER BY response_time_ms) AS p99
		FROM check_results
		WHERE monitor_id = $1
		  AND checked_at >= $2
	`, monitorID, since).Scan(
		&stats.LatestSuccess,
		&stats.TotalChecks,
		&stats.FailedChecks,
		&avg,
		&p50,
		&p95,
		&p99,
	)
	if err != nil {
		return CheckStats{}, observeDBError(err)
	}
	stats.AverageLatencyMs = round2(derefFloat(avg))
	stats.P50LatencyMs = round2(derefFloat(p50))
	stats.P95LatencyMs = round2(derefFloat(p95))
	stats.P99LatencyMs = round2(derefFloat(p99))
	return stats, nil
}

func derefFloat(value *float64) float64 {
	if value == nil {
		return 0
	}
	return *value
}

func scanCheckResult(row scanner) (*models.CheckResult, error) {
	var result models.CheckResult
	var jobID *uuid.UUID
	err := row.Scan(
		&result.ID,
		&jobID,
		&result.MonitorID,
		&result.StatusCode,
		&result.ResponseTimeMs,
		&result.Success,
		&result.ErrorMessage,
		&result.CheckedAt,
	)
	if err != nil {
		return nil, err
	}
	if jobID != nil {
		result.JobID = *jobID
	}
	return &result, nil
}

func nullableUUID(id uuid.UUID) any {
	if id == uuid.Nil {
		return nil
	}
	return id
}
