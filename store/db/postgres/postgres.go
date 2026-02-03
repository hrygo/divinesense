package postgres

import (
	"context"
	"database/sql"
	"log"
	"time"

	// Import the PostgreSQL driver.
	_ "github.com/lib/pq"
	"github.com/pkg/errors"

	"github.com/hrygo/divinesense/internal/profile"
	"github.com/hrygo/divinesense/store"
)

// ============================================================================
// POSTGRESQL SUPPORT (Production - Full Support)
// ============================================================================
// PostgreSQL is the PRIMARY database for production use.
//
// All features are fully supported:
// - Complete CRUD operations
// - Vector search (pgvector extension)
// - Full-text search (ts_vector, BM25)
// - Hybrid search (vector + BM25 with RRF fusion)
// - Advanced AI features (reranking)
// - Concurrent writes
// - Complex migrations
//
// When adding new features, PostgreSQL is the reference implementation.
// ============================================================================

type DB struct {
	db      *sql.DB
	profile *profile.Profile
}

func NewDB(profile *profile.Profile) (store.Driver, error) {
	if profile == nil {
		return nil, errors.New("profile is nil")
	}

	// Set default DSN if not provided (matching .env.example defaults)
	dsn := profile.DSN
	if dsn == "" {
		dsn = "postgres://divinesense:divinesense@localhost:25432/divinesense?sslmode=disable"
	}

	// Open the PostgreSQL connection
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		log.Printf("Failed to open database: %s", err)
		return nil, errors.Wrapf(err, "failed to open database: %s", dsn)
	}

	// Configure connection pool for single-user personal assistant
	// Optimized for low resource usage while maintaining responsiveness
	db.SetMaxOpenConns(5)                   // Single-user: max 5 concurrent connections
	db.SetMaxIdleConns(2)                   // Keep 2 idle connections ready for instant response
	db.SetConnMaxLifetime(2 * time.Hour)    // Personal use: longer lifetime, less churn
	db.SetConnMaxIdleTime(15 * time.Minute) // Close idle connections after 15min

	// Verify connection is working before returning
	if err := db.Ping(); err != nil {
		log.Printf("Failed to ping database: %s", err)
		return nil, errors.Wrap(err, "failed to ping database")
	}

	var driver store.Driver = &DB{
		db:      db,
		profile: profile,
	}

	// Return the DB struct
	return driver, nil
}

func (d *DB) GetDB() *sql.DB {
	return d.db
}

func (d *DB) Close() error {
	return d.db.Close()
}

func (d *DB) IsInitialized(ctx context.Context) (bool, error) {
	var exists bool
	err := d.db.QueryRowContext(ctx, "SELECT EXISTS (SELECT 1 FROM information_schema.tables WHERE table_catalog = current_database() AND table_name = 'memo' AND table_type = 'BASE TABLE')").Scan(&exists)
	if err != nil {
		return false, errors.Wrap(err, "failed to check if database is initialized")
	}
	return exists, nil
}

// AgentStatsStore returns the agent statistics store interface.
func (d *DB) AgentStatsStore() store.AgentStatsStore {
	return d
}

// SecurityAuditStore returns the security audit store interface.
func (d *DB) SecurityAuditStore() store.SecurityAuditStore {
	return d
}
