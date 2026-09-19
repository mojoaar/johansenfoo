package db

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"
	"time"
)

func TestOpenAppliesPragmasToEveryConnection(t *testing.T) {
	ctx := context.Background()
	d, err := Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer d.Close()

	d.SetMaxOpenConns(4)
	d.SetMaxIdleConns(4)

	var conns []*sql.Conn
	for i := 0; i < 4; i++ {
		c, err := d.Conn(ctx)
		if err != nil {
			t.Fatalf("Conn %d: %v", i, err)
		}
		defer c.Close()
		conns = append(conns, c)
	}

	for i, c := range conns {
		var foreignKeys, busyTimeout int
		if err := c.QueryRowContext(ctx, `PRAGMA foreign_keys`).Scan(&foreignKeys); err != nil {
			t.Fatalf("conn %d: read foreign_keys: %v", i, err)
		}
		if err := c.QueryRowContext(ctx, `PRAGMA busy_timeout`).Scan(&busyTimeout); err != nil {
			t.Fatalf("conn %d: read busy_timeout: %v", i, err)
		}
		if foreignKeys != 1 {
			t.Errorf("conn %d: foreign_keys = %d, want 1", i, foreignKeys)
		}
		if busyTimeout != 5000 {
			t.Errorf("conn %d: busy_timeout = %d, want 5000", i, busyTimeout)
		}
	}
}

func TestMigrateIsIdempotent(t *testing.T) {
	path := filepath.Join(t.TempDir(), "test.db")
	d, err := Open(path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer d.Close()

	if err := Migrate(d); err != nil {
		t.Fatalf("first Migrate: %v", err)
	}
	if err := Migrate(d); err != nil {
		t.Fatalf("second Migrate: %v", err)
	}

	var n int
	if err := d.QueryRow(`SELECT COUNT(*) FROM schema_migrations`).Scan(&n); err != nil {
		t.Fatalf("count migrations: %v", err)
	}
	if n == 0 {
		t.Error("schema_migrations is empty, want at least one row")
	}
}

func TestMigrationAppliedAtIsRFC3339UTC(t *testing.T) {
	d, err := Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer d.Close()
	if err := Migrate(d); err != nil {
		t.Fatalf("Migrate: %v", err)
	}

	var appliedAt string
	if err := d.QueryRow(
		`SELECT applied_at FROM schema_migrations ORDER BY version LIMIT 1`,
	).Scan(&appliedAt); err != nil {
		t.Fatalf("read applied_at: %v", err)
	}

	ts, err := time.Parse(time.RFC3339, appliedAt)
	if err != nil {
		t.Fatalf("applied_at %q is not RFC 3339: %v", appliedAt, err)
	}
	if ts.Location() != time.UTC {
		t.Errorf("applied_at %q is not UTC", appliedAt)
	}
}

func TestSchemaHasCoreTables(t *testing.T) {
	d, err := Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer d.Close()
	if err := Migrate(d); err != nil {
		t.Fatalf("Migrate: %v", err)
	}

	for _, table := range []string{
		"profile", "social_link", "project", "experience",
		"skill", "settings", "theme", "session", "schema_migrations",
		"post", "tag", "post_tag",
	} {
		var name string
		err := d.QueryRow(
			`SELECT name FROM sqlite_master WHERE type='table' AND name=?`, table,
		).Scan(&name)
		if err != nil {
			t.Errorf("table %q missing: %v", table, err)
		}
	}
}
