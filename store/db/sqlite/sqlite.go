package sqlite

import (
	"context"
	"database/sql"
	"time"

	"github.com/pkg/errors"

	// Import the SQLite driver.
	_ "modernc.org/sqlite"

	"github.com/hrygo/divinesense/internal/profile"
	"github.com/hrygo/divinesense/store"
)

// ============================================================================
// SQLITE SUPPORT POLICY
// ============================================================================
// SQLite is supported on a BEST-EFFORT basis for development and testing only.
//
// Supported Features (High ROI):
// - Basic CRUD operations
// - Simple queries
// - Single-user instances
//
// NOT Supported (Low ROI / High Complexity):
// - Concurrent writes (SQLite limitation)
// - Full-text search (BM25, hybrid search)
// - Advanced AI features (reranking)
// - Complex migrations
//
// When adding new features to SQLite:
// 1. Only implement if the ROI is high (low complexity, high value)
// 2. Prefer returning a clear error over partial/broken implementation
// 3. Add a comment explaining what is NOT supported
// ============================================================================

type DB struct {
	db      *sql.DB
	profile *profile.Profile
}

// NewDB opens a database specified by its database driver name and a
// driver-specific data source name, usually consisting of at least a
// database name and connection information.
func NewDB(profile *profile.Profile) (store.Driver, error) {
	// Ensure a DSN is set before attempting to open the database.
	if profile.DSN == "" {
		return nil, errors.New("dsn required")
	}

	// Connect to the database with some sane settings:
	// - No shared-cache: it's obsolete; WAL journal mode is a better solution.
	// - No foreign key constraints: it's currently disabled by default, but it's a
	// good practice to be explicit and prevent future surprises on SQLite upgrades.
	// - Journal mode set to WAL: it's the recommended journal mode for most applications
	// as it prevents locking issues.
	//
	// Notes:
	// - When using the `modernc.org/sqlite` driver, each pragma must be prefixed with `_pragma=`.
	//
	// References:
	// - https://pkg.go.dev/modernc.org/sqlite#Driver.Open
	// - https://www.sqlite.org/sharedcache.html
	// - https://www.sqlite.org/pragma.html
	sqliteDB, err := sql.Open("sqlite", profile.DSN+"?_pragma=foreign_keys(0)&_pragma=busy_timeout(10000)&_pragma=journal_mode(WAL)")
	if err != nil {
		return nil, errors.Wrapf(err, "failed to open db with dsn: %s", profile.DSN)
	}

	// Configure connection pool for single-user SQLite with WAL mode
	// SQLite handles concurrency differently; these settings optimize for local usage
	sqliteDB.SetMaxOpenConns(1)    // SQLite: single connection is optimal with WAL
	sqliteDB.SetMaxIdleConns(1)    // Keep the single connection ready
	sqliteDB.SetConnMaxLifetime(0) // No lifetime limit (local file, no network)
	sqliteDB.SetConnMaxIdleTime(0) // No idle timeout (personal use, always ready)

	driver := DB{db: sqliteDB, profile: profile}

	return &driver, nil
}

func (d *DB) GetDB() *sql.DB {
	return d.db
}

func (d *DB) Close() error {
	return d.db.Close()
}

func (d *DB) IsInitialized(ctx context.Context) (bool, error) {
	// Check if the database is initialized by checking if the memo table exists.
	var exists bool
	err := d.db.QueryRowContext(ctx, "SELECT EXISTS(SELECT 1 FROM sqlite_master WHERE type='table' AND name='memo')").Scan(&exists)
	if err != nil {
		return false, errors.Wrap(err, "failed to check if database is initialized")
	}
	return exists, nil
}

// AgentStatsStore returns the agent statistics store interface.
// SQLite does not support AI features (see #9).
func (d *DB) AgentStatsStore() store.AgentStatsStore {
	return &sqliteAgentStatsStore{db: d.db}
}

// SecurityAuditStore returns the security audit store interface.
// SQLite does not support AI features (see #9).
func (d *DB) SecurityAuditStore() store.SecurityAuditStore {
	return &sqliteSecurityAuditStore{db: d.db}
}

// sqliteAgentStatsStore is a no-op implementation for SQLite.
// AI features require PostgreSQL; see issue #9 for SQLite AI support research.
type sqliteAgentStatsStore struct {
	db *sql.DB
}

func (s *sqliteAgentStatsStore) SaveSessionStats(ctx context.Context, stats *store.AgentSessionStats) error {
	return errors.New("agent session stats not supported in SQLite (use PostgreSQL for AI features)")
}

func (s *sqliteAgentStatsStore) GetSessionStats(ctx context.Context, sessionID string) (*store.AgentSessionStats, error) {
	return nil, errors.New("agent session stats not supported in SQLite (use PostgreSQL for AI features)")
}

func (s *sqliteAgentStatsStore) ListSessionStats(ctx context.Context, userID int32, limit, offset int) ([]*store.AgentSessionStats, int64, error) {
	return nil, 0, errors.New("agent session stats not supported in SQLite (use PostgreSQL for AI features)")
}

func (s *sqliteAgentStatsStore) GetDailyCostUsage(ctx context.Context, userID int32, startDate, endDate time.Time) (float64, error) {
	return 0, errors.New("agent session stats not supported in SQLite (use PostgreSQL for AI features)")
}

func (s *sqliteAgentStatsStore) GetCostStats(ctx context.Context, userID int32, days int) (*store.CostStats, error) {
	return nil, errors.New("agent session stats not supported in SQLite (use PostgreSQL for AI features)")
}

func (s *sqliteAgentStatsStore) GetUserCostSettings(ctx context.Context, userID int32) (*store.UserCostSettings, error) {
	return nil, errors.New("agent session stats not supported in SQLite (use PostgreSQL for AI features)")
}

func (s *sqliteAgentStatsStore) SetUserCostSettings(ctx context.Context, settings *store.UserCostSettings) error {
	return errors.New("agent session stats not supported in SQLite (use PostgreSQL for AI features)")
}

// sqliteSecurityAuditStore is a no-op implementation for SQLite.
type sqliteSecurityAuditStore struct {
	db *sql.DB
}

func (s *sqliteSecurityAuditStore) LogSecurityEvent(ctx context.Context, event *store.SecurityAuditEvent) error {
	return errors.New("security audit logging not supported in SQLite (use PostgreSQL for AI features)")
}

func (s *sqliteSecurityAuditStore) ListSecurityEvents(ctx context.Context, userID int32, limit, offset int) ([]*store.SecurityAuditEvent, int64, error) {
	return nil, 0, errors.New("security audit logging not supported in SQLite (use PostgreSQL for AI features)")
}

func (s *sqliteSecurityAuditStore) ListSecurityEventsByRisk(ctx context.Context, userID int32, riskLevel string, limit, offset int) ([]*store.SecurityAuditEvent, int64, error) {
	return nil, 0, errors.New("security audit logging not supported in SQLite (use PostgreSQL for AI features)")
}

// ============================================================================
// AIBlock Methods (Unified Block Model)
// ============================================================================
// AI features require PostgreSQL; see issue #9 for SQLite AI support research.
// These methods return errors for SQLite to prevent silent failures.
// ============================================================================

func (d *DB) CreateAIBlock(ctx context.Context, create *store.CreateAIBlock) (*store.AIBlock, error) {
	return nil, errors.New("AIBlock not supported in SQLite (use PostgreSQL for AI features)")
}

func (d *DB) GetAIBlock(ctx context.Context, id int64) (*store.AIBlock, error) {
	return nil, errors.New("AIBlock not supported in SQLite (use PostgreSQL for AI features)")
}

func (d *DB) ListAIBlocks(ctx context.Context, find *store.FindAIBlock) ([]*store.AIBlock, error) {
	return nil, errors.New("AIBlock not supported in SQLite (use PostgreSQL for AI features)")
}

func (d *DB) UpdateAIBlock(ctx context.Context, update *store.UpdateAIBlock) (*store.AIBlock, error) {
	return nil, errors.New("AIBlock not supported in SQLite (use PostgreSQL for AI features)")
}

func (d *DB) DeleteAIBlock(ctx context.Context, id int64) error {
	return errors.New("AIBlock not supported in SQLite (use PostgreSQL for AI features)")
}

func (d *DB) AppendUserInput(ctx context.Context, blockID int64, input store.UserInput) error {
	return errors.New("AIBlock not supported in SQLite (use PostgreSQL for AI features)")
}

func (d *DB) AppendEvent(ctx context.Context, blockID int64, event store.BlockEvent) error {
	return errors.New("AIBlock not supported in SQLite (use PostgreSQL for AI features)")
}

func (d *DB) AppendEventsBatch(ctx context.Context, blockID int64, events []store.BlockEvent) error {
	return errors.New("AIBlock not supported in SQLite (use PostgreSQL for AI features)")
}

func (d *DB) UpdateAIBlockStatus(ctx context.Context, blockID int64, status store.AIBlockStatus) error {
	return errors.New("AIBlock not supported in SQLite (use PostgreSQL for AI features)")
}

func (d *DB) GetLatestAIBlock(ctx context.Context, conversationID int32) (*store.AIBlock, error) {
	return nil, errors.New("AIBlock not supported in SQLite (use PostgreSQL for AI features)")
}

func (d *DB) GetPendingAIBlocks(ctx context.Context) ([]*store.AIBlock, error) {
	return nil, errors.New("AIBlock not supported in SQLite (use PostgreSQL for AI features)")
}

func (d *DB) CreateAIBlockWithRound(ctx context.Context, create *store.CreateAIBlock) (*store.AIBlock, error) {
	return nil, errors.New("AIBlock not supported in SQLite (use PostgreSQL for AI features)")
}

func (d *DB) CompleteBlock(ctx context.Context, blockID int64, assistantContent string, sessionStats *store.SessionStats) error {
	return errors.New("AIBlock not supported in SQLite (use PostgreSQL for AI features)")
}
