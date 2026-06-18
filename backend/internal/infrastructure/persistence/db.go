// Package persistence provides database connection management
// and migration execution for the SQLite database.
//
// WHY a separate package: Database setup is infrastructure concern.
// It should be decoupled from business logic. This package opens
// the connection, configures SQLite settings, and runs migrations —
// everything the application needs before it can serve requests.
package persistence

import (
	"database/sql"
	"fmt"
	"os"

	// We use the mattn/go-sqlite3 driver, which requires CGO.
	// The underscore import is a Go idiom: we import it for its
	// side effect (registering the "sqlite3" driver with database/sql),
	// not to call any of its functions directly.
	// WHY mattn/go-sqlite3: It's the most widely-used SQLite driver
	// for Go, supports WAL mode, and is well-tested.
	_ "github.com/mattn/go-sqlite3"
)

// Open creates a connection pool to a SQLite database and configures
// it for optimal concurrent access.
//
// Parameters:
//   - dsn: The database connection string. For SQLite, this is the
//     file path. Use ":memory:" for an in-memory database (tests).
//
// Returns:
//   - *sql.DB: A connection pool (safe for concurrent use by multiple goroutines)
//   - error: If the database cannot be opened or configured
//
// WHY WAL mode: SQLite's default journal mode (rollback journal)
// blocks readers when a writer is active. WAL (Write-Ahead Logging)
// allows concurrent reads AND writes, which is critical for our
// WebSocket-broadcasting, sensor-ingesting server. Without WAL,
// sensor data ingestion would block history queries.
func Open(dsn string) (*sql.DB, error) {
	db, err := sql.Open("sqlite3", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// SQLite pragmas — these are SQLite-specific settings.
	// We set them as exec statements because sql.Open doesn't
	// actually connect; we need to execute these on the connection.

	// WAL mode: enables concurrent reads + writes.
	if err := executePragma(db, "PRAGMA journal_mode=WAL"); err != nil {
		return nil, fmt.Errorf("failed to set WAL mode: %w", err)
	}

	// Busy timeout: how long SQLite waits if the database is locked.
	// 5 seconds is generous — prevents "database is locked" errors
	// when multiple goroutines write concurrently.
	if err := executePragma(db, "PRAGMA busy_timeout=5000"); err != nil {
		return nil, fmt.Errorf("failed to set busy timeout: %w", err)
	}

	// Foreign keys: SQLite doesn't enforce FK constraints by default!
	// We must enable this explicitly. Without it, inserting sensor_data
	// with a non-existent vehicle_id would silently succeed.
	if err := executePragma(db, "PRAGMA foreign_keys=ON"); err != nil {
		return nil, fmt.Errorf("failed to enable foreign keys: %w", err)
	}

	// Connection pool settings:
	// - MaxOpenConns: 1 for SQLite (only one writer at a time; WAL allows
	//   concurrent readers but Go's connection pool doesn't know this,
	//   so we limit to 1 to avoid "database is locked" errors).
	// - MaxIdleConns: Keep 1 connection alive when idle.
	// WHY 1: SQLite is a file-based database, not a network server.
	// Multiple concurrent connections don't help — they just create
	// lock contention. WAL mode gives us read concurrency for free.
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)

	return db, nil
}

// executePragma runs a single PRAGMA statement on the database.
// This is a small helper to avoid repeating db.Exec + error handling.
func executePragma(db *sql.DB, pragma string) error {
	_, err := db.Exec(pragma)
	return err
}

// RunMigrations reads and executes SQL migration files from the given
// directory. Migrations run in alphabetical order (001_init.sql first).
//
// WHY simple file-based migrations: For a 3-day test, we don't need
// a migration framework like golang-migrate or goose. Just read the
// SQL file and execute it. This keeps the dependency count low.
//
// IMPORTANT: This is NOT a production-grade migration system.
// It doesn't track which migrations have run, and it uses
// "CREATE TABLE IF NOT EXISTS" for idempotency. Good enough for now.
func RunMigrations(db *sql.DB, migrationsDir string) error {
	// Read the initial migration file.
	// We could make this more sophisticated (walk the directory, order by name),
	// but for now there's only one migration file.
	migrationFile := migrationsDir + "/001_init.sql"
	sqlBytes, err := os.ReadFile(migrationFile)
	if err != nil {
		return fmt.Errorf("failed to read migration file %s: %w", migrationFile, err)
	}

	// Execute the entire SQL file as one statement.
	// SQLite's driver supports multiple statements in one Exec call.
	_, err = db.Exec(string(sqlBytes))
	if err != nil {
		return fmt.Errorf("failed to execute migration: %w", err)
	}

	return nil
}
