package database

import (
	"context"
	"embed"
	"fmt"
	"io/fs"
	"log"
	"path"
	"sort"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

//go:embed migrations/*.sql
var migrationFiles embed.FS

// Tables that an older Pulse run may have created without recording the
// matching migration. We never drop these; we only record the migration so
// it is not applied a second time.
var tablesCreatedByMigration = map[string]string{
	"000001_create_monitors.up.sql": "monitors",
}

func Migrate(ctx context.Context, pool *pgxpool.Pool) error {
	if err := ensureMigrationsTable(ctx, pool); err != nil {
		return err
	}

	entries, err := listUpMigrations()
	if err != nil {
		return err
	}

	for _, filename := range entries {
		if err := applyMigration(ctx, pool, filename); err != nil {
			return err
		}
	}

	return nil
}

func ensureMigrationsTable(ctx context.Context, pool *pgxpool.Pool) error {
	_, err := pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			filename TEXT PRIMARY KEY,
			applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)
	`)
	if err != nil {
		return fmt.Errorf("create schema_migrations: %w", err)
	}
	return nil
}

func listUpMigrations() ([]string, error) {
	entries, err := fs.Glob(migrationFiles, "migrations/*.up.sql")
	if err != nil {
		return nil, fmt.Errorf("list migrations: %w", err)
	}
	sort.Strings(entries)
	return entries, nil
}

func applyMigration(ctx context.Context, pool *pgxpool.Pool, filename string) error {
	base := path.Base(filename)

	tx, err := pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin migration %s: %w", base, err)
	}
	defer tx.Rollback(ctx)

	// Serialize migrators so two API processes cannot apply the same file.
	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(87211473)`); err != nil {
		return fmt.Errorf("lock migrations: %w", err)
	}

	applied, err := migrationIsRecorded(ctx, tx, base)
	if err != nil {
		return err
	}
	if applied {
		log.Printf("migration %s already applied, skipping", base)
		return tx.Commit(ctx)
	}

	// The monitors table can exist from an earlier run that did not record
	// 000001. Record it instead of running CREATE TABLE again.
	alreadyPresent, err := migrationAlreadyPresent(ctx, tx, base)
	if err != nil {
		return err
	}
	if alreadyPresent {
		if err := recordMigration(ctx, tx, base); err != nil {
			return err
		}
		log.Printf("migration %s already present in the database, recording it and skipping", base)
		return tx.Commit(ctx)
	}

	sqlBytes, err := migrationFiles.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("read migration %s: %w", base, err)
	}

	log.Printf("applying migration %s", base)
	// Migration files may contain more than one SQL statement (table + indexes).
	if err := tx.Conn().PgConn().Exec(ctx, string(sqlBytes)).Close(); err != nil {
		return fmt.Errorf("apply migration %s: %w", base, err)
	}

	if err := recordMigration(ctx, tx, base); err != nil {
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit migration %s: %w", base, err)
	}

	log.Printf("recorded migration %s in schema_migrations", base)
	return nil
}

func migrationIsRecorded(ctx context.Context, tx pgx.Tx, filename string) (bool, error) {
	var recorded bool
	err := tx.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM schema_migrations WHERE filename = $1
		)
	`, filename).Scan(&recorded)
	if err != nil {
		return false, fmt.Errorf("check migration %s: %w", filename, err)
	}
	return recorded, nil
}

func migrationAlreadyPresent(ctx context.Context, tx pgx.Tx, filename string) (bool, error) {
	tableName, ok := tablesCreatedByMigration[filename]
	if !ok {
		return false, nil
	}

	var exists bool
	err := tx.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1
			FROM information_schema.tables
			WHERE table_schema = 'public' AND table_name = $1
		)
	`, tableName).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("check existing table for %s: %w", filename, err)
	}
	return exists, nil
}

func recordMigration(ctx context.Context, tx pgx.Tx, filename string) error {
	_, err := tx.Exec(ctx, `
		INSERT INTO schema_migrations (filename) VALUES ($1)
	`, filename)
	if err != nil {
		return fmt.Errorf("record migration %s: %w", filename, err)
	}
	return nil
}
