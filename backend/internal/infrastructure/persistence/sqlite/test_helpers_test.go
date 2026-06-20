package sqlite

import (
	"database/sql"
	"path/filepath"
	"runtime"
	"testing"

	_ "github.com/mattn/go-sqlite3"

	"github.com/francisco/fleet-monitor/internal/infrastructure/persistence"
)

// setupTestDB creates an in-memory SQLite database and runs migrations.
// The database is automatically closed when the test finishes.
func setupTestDB(t *testing.T) *sql.DB {
	t.Helper()

	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("failed to open in-memory database: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	// Locate the migrations directory relative to this test file.
	// Path: backend/internal/infrastructure/persistence/sqlite/ → backend/migrations/
	// That's 4 directories up from this test file.
	_, filename, _, _ := runtime.Caller(0)
	dir := filepath.Dir(filename)
	migrationsDir := filepath.Join(dir, "..", "..", "..", "..", "migrations")

	// Use the shared RunMigrations so tests see the exact same logic
	// as production — including migration tracking.
	if err := persistence.RunMigrations(db, migrationsDir); err != nil {
		t.Fatalf("failed to run migrations: %v", err)
	}

	return db
}