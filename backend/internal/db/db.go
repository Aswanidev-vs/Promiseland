// Package db opens SQLite and applies embedded, versioned migrations.
package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"path/filepath"
	"sort"
	"strings"
	"time"

	migrations "github.com/Aswanidev-vs/promiseland/backend/migrations"
	_ "modernc.org/sqlite" // pure-Go SQLite driver, registered as "sqlite"
)

const driverName = "sqlite"

const createMigrationsTable = `CREATE TABLE IF NOT EXISTS schema_migrations (
	version TEXT PRIMARY KEY,
	applied_at TEXT NOT NULL DEFAULT (datetime('now'))
)`

type migration struct {
	version string
	sql     string
}

// Open opens the SQLite file with the pragmas required by the project
// (foreign keys, WAL, busy timeout) and configures the pool for a single
// writer. It pings the database before returning so startup fails clearly
// when the path or driver configuration is invalid.
func Open(path string) (*sql.DB, error) {
	if strings.TrimSpace(path) == "" {
		return nil, fmt.Errorf("db: database path must not be empty")
	}

	// Normalise separators: SQLite URIs use forward slashes on every platform.
	uri := "file:" + filepath.ToSlash(strings.TrimSpace(path)) +
		"?_pragma=foreign_keys(1)&_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)"

	database, err := sql.Open(driverName, uri)
	if err != nil {
		return nil, fmt.Errorf("db: open %q: %w", path, err)
	}

	// SQLite is used as a single-writer store; one connection also guarantees
	// that connection-scoped pragmas (foreign_keys, busy_timeout) always apply.
	database.SetMaxOpenConns(1)
	database.SetMaxIdleConns(1)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := database.PingContext(ctx); err != nil {
		_ = database.Close()
		return nil, fmt.Errorf("db: ping %q: %w", path, err)
	}
	return database, nil
}

// Migrate applies every embedded *.sql migration in filename order. Each
// migration runs in its own transaction and is recorded in schema_migrations,
// so repeated calls are safe (idempotent).
func Migrate(database *sql.DB, logger *slog.Logger) error {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	return migrate(ctx, database, logger)
}

func migrate(ctx context.Context, database *sql.DB, logger *slog.Logger) error {
	if database == nil {
		return fmt.Errorf("db: migrate: nil database")
	}
	if _, err := database.ExecContext(ctx, createMigrationsTable); err != nil {
		return fmt.Errorf("db: create schema_migrations: %w", err)
	}

	entries, err := fs.ReadDir(migrations.FS, ".")
	if err != nil {
		return fmt.Errorf("db: read embedded migrations: %w", err)
	}
	pending := make([]migration, 0, len(entries))
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".sql") {
			continue
		}
		content, err := migrations.FS.ReadFile(name)
		if err != nil {
			return fmt.Errorf("db: read migration %s: %w", name, err)
		}
		pending = append(pending, migration{version: name, sql: string(content)})
	}
	sort.Slice(pending, func(i, j int) bool { return pending[i].version < pending[j].version })

	for _, m := range pending {
		var applied int
		err := database.QueryRowContext(ctx, `SELECT 1 FROM schema_migrations WHERE version = ?`, m.version).Scan(&applied)
		if err == nil {
			continue // already applied
		}
		if !errors.Is(err, sql.ErrNoRows) {
			return fmt.Errorf("db: check migration %s: %w", m.version, err)
		}
		if err := apply(ctx, database, m); err != nil {
			return fmt.Errorf("db: apply migration %s: %w", m.version, err)
		}
		if logger != nil {
			logger.Info("database migration applied", "version", m.version)
		}
	}
	return nil
}

func apply(ctx context.Context, database *sql.DB, m migration) error {
	tx, err := database.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	if strings.TrimSpace(m.sql) != "" {
		if _, err := tx.ExecContext(ctx, m.sql); err != nil {
			return err
		}
	}
	if _, err := tx.ExecContext(ctx,
		`INSERT INTO schema_migrations (version, applied_at) VALUES (?, datetime('now'))`, m.version,
	); err != nil {
		return err
	}
	return tx.Commit()
}
