package db

import (
	"database/sql"
	"io"
	"log/slog"
	"path/filepath"
	"testing"
)

func TestOpenAndMigrate(t *testing.T) {
	database, err := Open(filepath.Join(t.TempDir(), "promisedland.db"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer func() { _ = database.Close() }()

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	if err := Migrate(database, logger); err != nil {
		t.Fatalf("first Migrate: %v", err)
	}

	for _, table := range []string{
		"schema_migrations", "users", "user_profiles", "sessions",
		"password_reset_tokens", "email_verification_tokens",
	} {
		var name string
		err := database.QueryRow(`SELECT name FROM sqlite_master WHERE type = 'table' AND name = ?`, table).Scan(&name)
		if err != nil {
			t.Errorf("table %s missing: %v", table, err)
		}
	}

	for _, index := range []string{"idx_sessions_user_id", "idx_users_email"} {
		var name string
		err := database.QueryRow(`SELECT name FROM sqlite_master WHERE type = 'index' AND name = ?`, index).Scan(&name)
		if err != nil {
			t.Errorf("index %s missing: %v", index, err)
		}
	}

	// Pragmas from the DSN.
	if got := queryString(t, database, `PRAGMA foreign_keys`); got != "1" {
		t.Errorf("foreign_keys = %q, want 1", got)
	}
	if got := queryString(t, database, `PRAGMA journal_mode`); got != "wal" {
		t.Errorf("journal_mode = %q, want wal", got)
	}
	if got := queryString(t, database, `PRAGMA busy_timeout`); got != "5000" {
		t.Errorf("busy_timeout = %q, want 5000", got)
	}

	// Idempotency: a second run must not re-apply or error.
	if err := Migrate(database, logger); err != nil {
		t.Fatalf("second Migrate: %v", err)
	}
	var count int
	if err := database.QueryRow(`SELECT COUNT(*) FROM schema_migrations`).Scan(&count); err != nil {
		t.Fatalf("count migrations: %v", err)
	}
	if count != 1 {
		t.Errorf("schema_migrations rows = %d, want 1", count)
	}
}

func TestForeignKeysAndCascade(t *testing.T) {
	database, err := Open(filepath.Join(t.TempDir(), "cascade.db"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer func() { _ = database.Close() }()
	if err := Migrate(database, nil); err != nil {
		t.Fatalf("Migrate: %v", err)
	}

	if _, err := database.Exec(`
		INSERT INTO users (id, email, password_hash, display_name)
		VALUES (1, 'test@example.com', 'not-a-real-hash', 'Test User')`); err != nil {
		t.Fatalf("insert user: %v", err)
	}
	if _, err := database.Exec(`
		INSERT INTO user_profiles (user_id, bio) VALUES (1, 'hello')`); err != nil {
		t.Fatalf("insert profile: %v", err)
	}
	if _, err := database.Exec(`
		INSERT INTO sessions (token_hash, user_id, expires_at)
		VALUES ('hash', 1, datetime('now', '+1 day'))`); err != nil {
		t.Fatalf("insert session: %v", err)
	}

	// FK enforcement: a session for a missing user must be rejected.
	if _, err := database.Exec(`
		INSERT INTO sessions (token_hash, user_id, expires_at)
		VALUES ('orphan', 999, datetime('now', '+1 day'))`); err == nil {
		t.Error("expected foreign key violation for orphan session")
	}

	if _, err := database.Exec(`DELETE FROM users WHERE id = 1`); err != nil {
		t.Fatalf("delete user: %v", err)
	}
	for _, table := range []string{"user_profiles", "sessions"} {
		var count int
		if err := database.QueryRow(`SELECT COUNT(*) FROM ` + table).Scan(&count); err != nil {
			t.Fatalf("count %s: %v", table, err)
		}
		if count != 0 {
			t.Errorf("%s rows after user delete = %d, want 0", table, count)
		}
	}
}

func TestOpenRejectsEmptyPath(t *testing.T) {
	if _, err := Open("  "); err == nil {
		t.Error("expected error for empty database path")
	}
}

func queryString(t *testing.T, database *sql.DB, query string) string {
	t.Helper()
	var value string
	if err := database.QueryRow(query).Scan(&value); err != nil {
		t.Fatalf("query %q: %v", query, err)
	}
	return value
}
