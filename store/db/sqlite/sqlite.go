package sqlite

import (
	"context"
	"database/sql"
	"log/slog"
	"time"

	"github.com/pkg/errors"

	// Import the SQLite driver with CGO support for sqlite-vec.
	_ "github.com/mattn/go-sqlite3"

	"github.com/hrygo/divinesense/internal/profile"
	"github.com/hrygo/divinesense/store"
)

// ============================================================================
// SQLITE SUPPORT POLICY
// ============================================================================
// SQLite is supported for development and client-side deployment.
//
// Supported Features:
// - Full AI features: vector search (via sqlite-vec), conversation persistence, episodic memory
// - Vector search: sqlite-vec extension with vec0_distance() for efficient similarity search
// - Full-text search: FTS5 (if available) with LIKE fallback
// - All AI agent capabilities: memo, schedule, amazing agents
//
// Implementation Notes:
// - Vectors stored in vec0 format for optimal performance
// - Similarity computed using sqlite-vec's vec0_distance_L2 function
// - JSONB fields replaced with TEXT (JSON strings)
// - Requires CGO for sqlite-vec extension
// - Performance suitable for large datasets (>100k vectors)
//
// When adding new features to SQLite:
// 1. Maintain feature parity with PostgreSQL where feasible
// 2. Document any performance limitations
// 3. Use mattn/go-sqlite3 with CGO for sqlite-vec support
// ============================================================================

type DB struct {
	db                 *sql.DB
	profile            *profile.Profile
	vecExtensionLoaded bool // Track if sqlite-vec extension is loaded
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
	// - Foreign key constraints: enabled for referential integrity
	// - Journal mode set to WAL: it's the recommended journal mode for most applications
	//   as it prevents locking issues.
	//
	// Notes:
	// - When using the `github.com/mattn/go-sqlite3` driver with CGO:
	//   - _pragma= syntax is not supported, use pragmas via EXEC instead
	//   - Load extensions enabled for sqlite-vec support
	//
	// References:
	// - https://pkg.go.dev/github.com/mattn/go-sqlite3
	// - https://www.sqlite.org/sharedcache.html
	// - https://www.sqlite.org/pragma.html
	sqliteDB, err := sql.Open("sqlite3", profile.DSN)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to open db with dsn: %s", profile.DSN)
	}

	// Configure SQLite pragmas
	pragmas := []string{
		"PRAGMA foreign_keys = ON",
		"PRAGMA journal_mode = WAL",
		"PRAGMA busy_timeout = 10000",
	}
	for _, pragma := range pragmas {
		if _, err := sqliteDB.Exec(pragma); err != nil {
			return nil, errors.Wrapf(err, "failed to set pragma: %s", pragma)
		}
	}

	// Enable extension loading (required for sqlite-vec)
	// Note: This must be called before any load_extension() calls
	if _, err := sqliteDB.Exec("PRAGMA enable_load_extension = true"); err != nil {
		return nil, errors.Wrap(err, "failed to enable extension loading")
	}

	// Load sqlite-vec extension for vector search
	// The extension is compiled into libvec0.a (static library)
	//
	// Build process:
	// 1. Download SQLite amalgamation and sqlite-vec source
	// 2. Compile as static library: ar rcs libvec0.a sqlite3.o sqlite-vec.o
	// 3. Link with Go binary via CGO
	//
	// Load locations tried:
	// - System install (pkg install, apt, etc.)
	// - Build directory (development)
	// - Embedded in binary (via go:embed)
	vecLoaded := false
	if err := loadVecExtension(sqliteDB); err != nil {
		// Log warning but don't fail - vector search will use fallback
		// This allows graceful degradation for development environments
		slog.Warn("sqlite-vec extension not loaded, vector search will use Go fallback",
			"error", err,
		)
		vecLoaded = false
	} else {
		vecLoaded = true
	}

	// Configure connection pool for single-user SQLite with WAL mode
	// SQLite handles concurrency differently; these settings optimize for local usage
	sqliteDB.SetMaxOpenConns(1)    // SQLite: single connection is optimal with WAL
	sqliteDB.SetMaxIdleConns(1)    // Keep the single connection ready
	sqliteDB.SetConnMaxLifetime(0) // No lifetime limit (local file, no network)
	sqliteDB.SetConnMaxIdleTime(0) // No idle timeout (personal use, always ready)

	driver := DB{
		db:                 sqliteDB,
		profile:            profile,
		vecExtensionLoaded: vecLoaded,
	}

	return &driver, nil
}

func (d *DB) GetDB() *sql.DB {
	return d.db
}

func (d *DB) GetVecExtensionLoaded() bool {
	return d.vecExtensionLoaded
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
	// TODO: Implement full AIBlock support for SQLite
	// For now, return empty list to prevent frontend errors
	// AIBlock is the new unified conversation model from main branch
	// Current SQLite implementation uses AIConversation instead
	return []*store.AIBlock{}, nil
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

// ========== Tree Branching Methods (tree-conversation-branching) ==========
// SQLite does not support AI features including tree branching (see #9).

func (d *DB) ForkBlock(ctx context.Context, parentID int64, reason string, replaceUserInputs []store.UserInput) (*store.AIBlock, error) {
	return nil, errors.New("AIBlock branching not supported in SQLite (use PostgreSQL for AI features)")
}

func (d *DB) ListChildBlocks(ctx context.Context, parentID int64) ([]*store.AIBlock, error) {
	return nil, errors.New("AIBlock branching not supported in SQLite (use PostgreSQL for AI features)")
}

func (d *DB) GetActivePath(ctx context.Context, conversationID int32) ([]*store.AIBlock, error) {
	return nil, errors.New("AIBlock branching not supported in SQLite (use PostgreSQL for AI features)")
}

func (d *DB) DeleteBranch(ctx context.Context, blockID int64, cascade bool) error {
	return errors.New("AIBlock branching not supported in SQLite (use PostgreSQL for AI features)")
}

func (d *DB) ArchiveInactiveBranches(ctx context.Context, conversationID int32, targetPath string, archivedAt int64) error {
	return errors.New("AIBlock branching not supported in SQLite (use PostgreSQL for AI features)")
}

// ============================================================================
// AIConversation Methods (Legacy API - maintained for compatibility)
// ============================================================================
// AIConversation is the legacy conversation model, superseded by AIBlock.
// These methods return errors for SQLite to prevent silent failures.
// ============================================================================

func (d *DB) CreateAIConversation(ctx context.Context, create *store.AIConversation) (*store.AIConversation, error) {
	return nil, errors.New("AIConversation not supported in SQLite (use PostgreSQL for AI features)")
}

func (d *DB) ListAIConversations(ctx context.Context, find *store.FindAIConversation) ([]*store.AIConversation, error) {
	return nil, errors.New("AIConversation not supported in SQLite (use PostgreSQL for AI features)")
}

func (d *DB) GetAIConversation(ctx context.Context, id int32) (*store.AIConversation, error) {
	return nil, errors.New("AIConversation not supported in SQLite (use PostgreSQL for AI features)")
}

func (d *DB) UpdateAIConversation(ctx context.Context, update *store.UpdateAIConversation) (*store.AIConversation, error) {
	return nil, errors.New("AIConversation not supported in SQLite (use PostgreSQL for AI features)")
}

func (d *DB) DeleteAIConversation(ctx context.Context, delete *store.DeleteAIConversation) error {
	return errors.New("AIConversation not supported in SQLite (use PostgreSQL for AI features)")
}

func (d *DB) ListAIConversationsBasic(ctx context.Context, find *store.FindAIConversation) ([]*store.AIConversation, error) {
	return nil, errors.New("AIConversation not supported in SQLite (use PostgreSQL for AI features)")
}

// ============================================================================
// EpisodicMemory Methods (NOT SUPPORTED - use PostgreSQL)
// ============================================================================

func (d *DB) CreateEpisodicMemory(ctx context.Context, create *store.EpisodicMemory) (*store.EpisodicMemory, error) {
	return nil, errors.New("episodic memory not supported in SQLite (use PostgreSQL for AI features)")
}

func (d *DB) ListEpisodicMemories(ctx context.Context, find *store.FindEpisodicMemory) ([]*store.EpisodicMemory, error) {
	return nil, errors.New("episodic memory not supported in SQLite (use PostgreSQL for AI features)")
}

func (d *DB) ListActiveUserIDs(ctx context.Context, cutoff time.Time) ([]int32, error) {
	return nil, errors.New("episodic memory not supported in SQLite (use PostgreSQL for AI features)")
}

func (d *DB) DeleteEpisodicMemory(ctx context.Context, delete *store.DeleteEpisodicMemory) error {
	return errors.New("episodic memory not supported in SQLite (use PostgreSQL for AI features)")
}

// ============================================================================
// UserPreferences Methods (NOT SUPPORTED - use PostgreSQL)
// ============================================================================

func (d *DB) UpsertUserPreferences(ctx context.Context, upsert *store.UpsertUserPreferences) (*store.UserPreferences, error) {
	return nil, errors.New("user preferences not supported in SQLite (use PostgreSQL for AI features)")
}

func (d *DB) GetUserPreferences(ctx context.Context, find *store.FindUserPreferences) (*store.UserPreferences, error) {
	return nil, errors.New("user preferences not supported in SQLite (use PostgreSQL for AI features)")
}

// ============================================================================
// AgentMetrics Methods (NOT SUPPORTED - use PostgreSQL)
// ============================================================================

func (d *DB) UpsertAgentMetrics(ctx context.Context, upsert *store.UpsertAgentMetrics) (*store.AgentMetrics, error) {
	return nil, errors.New("agent metrics not supported in SQLite (use PostgreSQL for AI features)")
}

func (d *DB) ListAgentMetrics(ctx context.Context, find *store.FindAgentMetrics) ([]*store.AgentMetrics, error) {
	return nil, errors.New("agent metrics not supported in SQLite (use PostgreSQL for AI features)")
}

func (d *DB) DeleteAgentMetrics(ctx context.Context, delete *store.DeleteAgentMetrics) error {
	return errors.New("agent metrics not supported in SQLite (use PostgreSQL for AI features)")
}

func (d *DB) UpsertToolMetrics(ctx context.Context, upsert *store.UpsertToolMetrics) (*store.ToolMetrics, error) {
	return nil, errors.New("tool metrics not supported in SQLite (use PostgreSQL for AI features)")
}

func (d *DB) ListToolMetrics(ctx context.Context, find *store.FindToolMetrics) ([]*store.ToolMetrics, error) {
	return nil, errors.New("tool metrics not supported in SQLite (use PostgreSQL for AI features)")
}

func (d *DB) DeleteToolMetrics(ctx context.Context, delete *store.DeleteToolMetrics) error {
	return errors.New("tool metrics not supported in SQLite (use PostgreSQL for AI features)")
}
