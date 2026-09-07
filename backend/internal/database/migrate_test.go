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
}
