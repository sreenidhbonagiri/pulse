package database

import (
	"path"
	"testing"
)

func TestListUpMigrationsIsOrdered(t *testing.T) {
	entries, err := listUpMigrations()
	if err != nil {
		t.Fatalf("listUpMigrations: %v", err)
	}
	if len(entries) == 0 {
		t.Fatal("expected at least one up migration")
	}

	for i := 1; i < len(entries); i++ {
		if entries[i-1] > entries[i] {
			t.Fatalf("migrations are not ordered: %v", entries)
		}
	}

	if path.Base(entries[0]) != "000001_create_monitors.up.sql" {
		t.Fatalf("first migration = %q", entries[0])
	}
	if len(entries) < 2 || path.Base(entries[1]) != "000002_create_check_results.up.sql" {
		t.Fatalf("second migration = %q", entries)
	}
	if len(entries) < 3 || path.Base(entries[2]) != "000003_add_monitors_next_check_at.up.sql" {
		t.Fatalf("third migration = %q", entries)
	}
	if len(entries) < 4 || path.Base(entries[3]) != "000004_add_check_results_job_id.up.sql" {
		t.Fatalf("fourth migration = %q", entries)
	}
	if len(entries) < 5 || path.Base(entries[4]) != "000005_create_incidents.up.sql" {
		t.Fatalf("fifth migration = %q", entries)
	}
}
