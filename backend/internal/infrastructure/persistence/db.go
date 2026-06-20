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
	"path/filepath"
	"sort"

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
// directory. Migrations run in alphabetical order (001_init.sql first,
// then 002_*.sql, etc.).
//
// Tracking: A schema_migrations table records which migrations have been
// applied. Each migration that executes successfully gets its filename
// recorded. On subsequent runs, already-applied migrations are skipped.
// This prevents errors from non-idempotent statements like
// ALTER TABLE ADD COLUMN on the second startup.
//
// WHY simple file-based migrations: For a 3-day test, we don't need
// a migration framework like golang-migrate or goose. File-based +
// tracking table is explicit, debuggable, and dependency-free.
func RunMigrations(db *sql.DB, migrationsDir string) error {
	// Ensure the tracking table exists before we check or record anything.
	if err := ensureMigrationsTable(db); err != nil {
		return fmt.Errorf("failed to create schema_migrations table: %w", err)
	}

	entries, err := os.ReadDir(migrationsDir)
	if err != nil {
		return fmt.Errorf("failed to read migrations directory %s: %w", migrationsDir, err)
	}

	// Collect .sql files and run them in alphabetical (filename) order.
	var files []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if filepath.Ext(name) != ".sql" {
			continue
		}
		files = append(files, name)
	}
	sort.Strings(files)

	for _, name := range files {
		// Check if this migration was already applied.
		var count int
		row := db.QueryRow("SELECT COUNT(1) FROM schema_migrations WHERE version = $1", name)
		if err := row.Scan(&count); err != nil {
			return fmt.Errorf("failed to check migration status for %s: %w", name, err)
		}
		if count > 0 {
			// Already applied — skip silently.
			continue
		}

		path := filepath.Join(migrationsDir, name)
		sqlBytes, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("failed to read migration file %s: %w", path, err)
		}

		// Run the migration inside a transaction so we can record it
		// atomically. If the SQL fails, the transaction rolls back and
		// the version is NOT recorded — the next startup will retry it.
		tx, err := db.Begin()
		if err != nil {
			return fmt.Errorf("failed to begin transaction for migration %s: %w", name, err)
		}

		if _, err := tx.Exec(string(sqlBytes)); err != nil {
			tx.Rollback()
			return fmt.Errorf("failed to execute migration %s: %w", name, err)
		}

		if _, err := tx.Exec(
			"INSERT INTO schema_migrations (version, applied_at) VALUES ($1, datetime('now'))",
			name,
		); err != nil {
			tx.Rollback()
			return fmt.Errorf("failed to record migration %s: %w", name, err)
		}

		if err := tx.Commit(); err != nil {
			return fmt.Errorf("failed to commit migration %s: %w", name, err)
		}
	}

	return nil
}

// ensureMigrationsTable creates the schema_migrations tracking table
// if it does not already exist. This is called before any migration
// checks or execution.
func ensureMigrationsTable(db *sql.DB) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version   TEXT PRIMARY KEY,
			applied_at TEXT NOT NULL
		)
	`)
	return err
}
