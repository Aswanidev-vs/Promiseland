// Package testutil provides small SQLite fixtures for R1 module tests.
// It is test-only support code and is never imported by the running server.
package testutil

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"io"
	"log/slog"
	"testing"

	"github.com/Aswanidev-vs/promiseland/backend/internal/db"
	"github.com/Aswanidev-vs/promiseland/backend/internal/shared"
	"github.com/Aswanidev-vs/promiseland/backend/internal/testutil"
)

// Deps opens a fresh migrated database in a temporary directory.
func Deps(t testing.TB) *shared.Deps {
	t.Helper()
	// TempDB closes the pool and retries directory removal: SQLite can still
	// hold its WAL sidecar open for a moment after Close, and t.TempDir's
	// removal fails the test on Windows when that happens.
	database, _ := testutil.TempDB(t, "test.db")
	if err := db.Migrate(database, nil); err != nil {
		t.Fatalf("migrate test database: %v", err)
	}
	return &shared.Deps{
		DB:  database,
		Log: slog.New(slog.NewTextHandler(io.Discard, nil)),
	}
}

// User inserts an active account with a session and returns (id, token).
func User(t testing.TB, d *shared.Deps, username string) (int64, string) {
	t.Helper()
	res, err := d.DB.Exec(`
		INSERT INTO users (email, password_hash, display_name, username)
		VALUES (?, 'hashed', ?, ?)`,
		username+"@example.test", username, username)
	if err != nil {
		t.Fatalf("insert user %s: %v", username, err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		t.Fatalf("user id: %v", err)
	}
	if _, err := d.DB.Exec(`INSERT INTO user_profiles (user_id, visibility) VALUES (?, 'public')`, id); err != nil {
		t.Fatalf("insert profile: %v", err)
	}
	token := fmt.Sprintf("session-%s-%d", username, id)
	sum := sha256.Sum256([]byte(token))
	if _, err := d.DB.Exec(`
		INSERT INTO sessions (token_hash, user_id, expires_at)
		VALUES (?, ?, datetime('now', '+1 day'))`, hex.EncodeToString(sum[:]), id); err != nil {
		t.Fatalf("insert session: %v", err)
	}
	return id, token
}

// Exec runs a statement, failing the test on error.
func Exec(t testing.TB, d *shared.Deps, query string, args ...any) {
	t.Helper()
	if _, err := d.DB.Exec(query, args...); err != nil {
		t.Fatalf("exec %q: %v", query, err)
	}
}

// QueryInt returns the first integer of a query, or 0 when absent.
func QueryInt(t testing.TB, d *shared.Deps, query string, args ...any) int {
	t.Helper()
	var value int
	if err := d.DB.QueryRow(query, args...).Scan(&value); err != nil && err != sql.ErrNoRows {
		t.Fatalf("query %q: %v", query, err)
	}
	return value
}
