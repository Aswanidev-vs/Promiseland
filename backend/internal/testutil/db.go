// Package testutil provides small helpers shared by backend tests.
package testutil

import (
	"database/sql"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Aswanidev-vs/promiseland/backend/internal/db"
)

// TempDB creates a temp directory and opens a SQLite database inside it,
// registering a cleanup that closes the pool and retries directory removal
// to survive Windows file-lock races with SQLite WAL sidecar files.
func TempDB(t testing.TB, name string) (*sql.DB, string) {
	t.Helper()
	dir, err := os.MkdirTemp("", "pl-test-")
	if err != nil {
		t.Fatalf("testutil: create temp dir: %v", err)
	}
	path := filepath.Join(dir, name)
	database, err := db.Open(path)
	if err != nil {
		_ = os.RemoveAll(dir)
		t.Fatalf("testutil: open %q: %v", path, err)
	}
	t.Cleanup(func() {
		_ = database.Close()
		for attempt := 0; attempt < 200; attempt++ {
			if err := os.RemoveAll(dir); err == nil {
				return
			}
			time.Sleep(10 * time.Millisecond)
		}
		t.Errorf("testutil: could not remove temp dir %s after 200 attempts", dir)
	})
	return database, path
}
