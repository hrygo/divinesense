package store

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"io/fs"
	"log/slog"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/pkg/errors"

	"github.com/hrygo/divinesense/internal/version"
	storepb "github.com/hrygo/divinesense/proto/gen/store"
)

// Migration System Overview:
//
// The migration system handles database schema versioning and upgrades.
// Schema version is stored in system_setting.
//
// Migration Flow:
// 1. preMigrate: Check if DB is initialized. If not, apply LATEST.sql
// 2. checkMinimumUpgradeVersion: Verify installation can be upgraded (reject pre-0.22 installations)
// 3. Migrate (prod mode): Apply incremental migrations from current to target version
// 4. Migrate (demo mode): Seed database with demo data
//
// Version Tracking:
// - New installations: Schema version set in system_setting immediately
// - Existing v0.22+ installations: Schema version tracked in system_setting
// - Pre-v0.22 installations: Must upgrade to v0.25.x first (migration_history → system_setting migration)
//
// Migration Files:
// - Location: store/migration/{driver}/migrate/{timestamp}_{description}.{up|down}.sql
// - Schema: store/migration/{driver}/schema/LATEST.sql (full schema for new installations)
// - Naming: {YYYY}{MM}{DD}{HH}{MM}{SS}_{snake_case_description}.{up|down}.sql
// - Ordering: Files sorted by timestamp, applied sequentially
// - Up/Down: Separate files for apply and rollback

//go:embed migration
var migrationFS embed.FS

//go:embed seed
var seedFS embed.FS

const (
	// MigrateFilePrefix is the prefix for migration files.
	// Migration files use timestamp format: YYYYMMDDHHMMSS_description.sql
	MigrateFilePrefix = ""
	// LatestSchemaFileName is the name of the latest schema file.
	// This file is used to initialize fresh installations with the current schema.
	LatestSchemaFileName = "LATEST.sql"
	// MigrateDirName is the subdirectory containing migration files.
	MigrateDirName = "migrate"
	// SchemaDirName is the subdirectory containing the base schema file.
	SchemaDirName = "schema"

	// defaultSchemaVersion is used when schema version is empty or not set.
	// This handles edge cases for old installations without version tracking.
	defaultSchemaVersion = "0.0.0"

	// Mode constants for profile mode.
	modeProd = "prod"
	modeDemo = "demo"
)

// getSchemaVersionOrDefault returns the schema version or default if empty.
// This ensures safe version comparisons and handles old installations.
func getSchemaVersionOrDefault(schemaVersion string) string {
	if schemaVersion == "" {
		return defaultSchemaVersion
	}
	return schemaVersion
}

// isVersionEmpty checks if the schema version is empty or the default value.
func isVersionEmpty(schemaVersion string) bool {
	return schemaVersion == "" || schemaVersion == defaultSchemaVersion
}

// shouldApplyMigration determines if a migration file should be applied.
// It checks if the file's version is between the current DB version and target version.
func shouldApplyMigration(fileVersion, currentDBVersion, targetVersion string) bool {
	currentDBVersionSafe := getSchemaVersionOrDefault(currentDBVersion)
	return version.IsVersionGreaterThan(fileVersion, currentDBVersionSafe) &&
		version.IsVersionGreaterOrEqualThan(targetVersion, fileVersion)
}

// validateMigrationFileName checks if a migration file follows the expected naming convention.
// Expected format: "YYYYMMDDHHMMSS_description.{up|down}.sql"
// Only .up.sql files should be applied; .down.sql files are for rollback.
func validateMigrationFileName(filename string) error {
	// Skip down migration files
	if strings.HasSuffix(filename, ".down.sql") {
		return errors.Errorf("down migration files should not be applied: %s", filename)
	}
	// Check for .up.sql extension
	if !strings.HasSuffix(filename, ".up.sql") {
		return errors.Errorf("migration file must have .up.sql extension: %s", filename)
	}
	// Extract the part before the first underscore
	baseName := strings.TrimSuffix(filename, ".up.sql")
	parts := strings.SplitN(baseName, "_", 2)
	if len(parts) < 2 {
		return errors.Errorf("migration filename must be in format YYYYMMDDHHMMSS_description.up.sql: %s", filename)
	}
	// Check if first part is a 14-digit timestamp
	timestamp := parts[0]
	if len(timestamp) != 14 {
		return errors.Errorf("migration timestamp must be 14 digits (YYYYMMDDHHMMSS): %s", filename)
	}
	if _, err := strconv.Atoi(timestamp); err != nil {
		return errors.Errorf("migration timestamp must be numeric: %s", filename)
	}
	return nil
}

// Migrate migrates the database schema to the latest version.
// It checks the current schema version and applies any necessary migrations.
// It also seeds the database with initial data if in demo mode.
func (s *Store) Migrate(ctx context.Context) error {
	if err := s.preMigrate(ctx); err != nil {
		return errors.Wrap(err, "failed to pre-migrate")
	}

	switch s.profile.Mode {
	case modeProd:
		instanceBasicSetting, err := s.GetInstanceBasicSetting(ctx)
		if err != nil {
			return errors.Wrap(err, "failed to get instance basic setting")
		}
		currentSchemaVersion, err := s.GetCurrentSchemaVersion()
		if err != nil {
			return errors.Wrap(err, "failed to get current schema version")
		}
		// Check for downgrade (but skip if schema version is empty - that means fresh/old installation)
		if !isVersionEmpty(instanceBasicSetting.SchemaVersion) && version.IsVersionGreaterThan(instanceBasicSetting.SchemaVersion, currentSchemaVersion) {
			slog.Error("cannot downgrade schema version",
				slog.String("databaseVersion", instanceBasicSetting.SchemaVersion),
				slog.String("currentVersion", currentSchemaVersion),
			)
			return errors.Errorf("cannot downgrade schema version from %s to %s", instanceBasicSetting.SchemaVersion, currentSchemaVersion)
		}
		// Apply migrations if needed (including when schema version is empty)
		if isVersionEmpty(instanceBasicSetting.SchemaVersion) || version.IsVersionGreaterThan(currentSchemaVersion, instanceBasicSetting.SchemaVersion) {
			if err := s.applyMigrations(ctx, instanceBasicSetting.SchemaVersion, currentSchemaVersion); err != nil {
				return errors.Wrap(err, "failed to apply migrations")
			}
		}
	case modeDemo:
		// In demo mode, we should seed the database.
		if err := s.seed(ctx); err != nil {
			return errors.Wrap(err, "failed to seed")
		}
	default:
		// For other modes (like dev), no special migration handling needed
	}
	return nil
}

// applyMigrations applies all necessary migration files between current and target schema versions.
// It runs all migrations in a single transaction for atomicity.
func (s *Store) applyMigrations(ctx context.Context, currentSchemaVersion, targetSchemaVersion string) error {
	filePaths, err := fs.Glob(migrationFS, fmt.Sprintf("%s*/*.sql", s.getMigrationBasePath()))
	if err != nil {
		return errors.Wrap(err, "failed to read migration files")
	}
	sort.Strings(filePaths)

	// Start a transaction to apply migrations atomically
	tx, err := s.driver.GetDB().Begin()
	if err != nil {
		return errors.Wrap(err, "failed to start transaction")
	}
	defer func() {
		_ = tx.Rollback()
	}()

	// Use safe version for comparison (handles empty version case)
	schemaVersionForComparison := getSchemaVersionOrDefault(currentSchemaVersion)
	if isVersionEmpty(currentSchemaVersion) {
		slog.Warn("schema version is empty, treating as default for migration comparison",
			slog.String("defaultVersion", defaultSchemaVersion))
	}

	slog.Info("start migration",
		slog.String("currentSchemaVersion", schemaVersionForComparison),
		slog.String("targetSchemaVersion", targetSchemaVersion))

	migrationsApplied := 0
	for _, filePath := range filePaths {
		fileSchemaVersion, err := s.getSchemaVersionOfMigrateScript(filePath)
		if err != nil {
			return errors.Wrap(err, "failed to get schema version of migrate script")
		}

		if shouldApplyMigration(fileSchemaVersion, currentSchemaVersion, targetSchemaVersion) {
			// Validate migration filename before applying
			filename := filepath.Base(filePath)
			if err := validateMigrationFileName(filename); err != nil {
				slog.Warn("migration file has invalid name but will be applied", slog.String("file", filePath), slog.String("error", err.Error()))
			}

			slog.Info("applying migration",
				slog.String("file", filePath),
				slog.String("version", fileSchemaVersion))

			bytes, err := migrationFS.ReadFile(filePath)
			if err != nil {
				return errors.Wrapf(err, "failed to read migration file: %s", filePath)
			}

			stmt := string(bytes)
			if err := s.execute(ctx, tx, stmt); err != nil {
				return errors.Wrapf(err, "failed to execute migration %s: %s", filePath, err)
			}
			migrationsApplied++
		}
	}

	if err := tx.Commit(); err != nil {
		return errors.Wrap(err, "failed to commit migration transaction")
	}

	slog.Info("migration completed", slog.Int("migrationsApplied", migrationsApplied))

	// Update schema version after successful migration
	if err := s.updateCurrentSchemaVersion(ctx, targetSchemaVersion); err != nil {
		return errors.Wrap(err, "failed to update current schema version")
	}

	return nil
}

// preMigrate checks if the database is initialized and applies the latest schema if not.
func (s *Store) preMigrate(ctx context.Context) error {
	initialized, err := s.driver.IsInitialized(ctx)
	if err != nil {
		return errors.Wrap(err, "failed to check if database is initialized")
	}

	if !initialized {
		filePath := s.getSchemaBasePath() + LatestSchemaFileName
		bytes, err := migrationFS.ReadFile(filePath)
		if err != nil {
			return errors.Errorf("failed to read latest schema file: %s", err)
		}
		// Start a transaction to apply the latest schema.
		tx, err := s.driver.GetDB().Begin()
		if err != nil {
			return errors.Wrap(err, "failed to start transaction")
		}
		defer func() {
			_ = tx.Rollback()
		}()
		slog.Info("initializing new database with latest schema", slog.String("file", filePath))
		if err := s.execute(ctx, tx, string(bytes)); err != nil {
			return errors.Errorf("failed to execute SQL file %s, err %s", filePath, err)
		}
		if err := tx.Commit(); err != nil {
			return errors.Wrap(err, "failed to commit transaction")
		}

		// Upsert current schema version to database.
		schemaVersion, err := s.GetCurrentSchemaVersion()
		if err != nil {
			return errors.Wrap(err, "failed to get current schema version")
		}
		slog.Info("database initialized successfully", slog.String("schemaVersion", schemaVersion))
		if err := s.updateCurrentSchemaVersion(ctx, schemaVersion); err != nil {
			return errors.Wrap(err, "failed to update current schema version")
		}
	}

	if s.profile.Mode == modeProd {
		if err := s.checkMinimumUpgradeVersion(ctx); err != nil {
			return err // Error message is already descriptive, don't wrap it
		}
	}
	return nil
}

func (s *Store) getMigrationBasePath() string {
	return fmt.Sprintf("migration/%s/%s/", s.profile.Driver, MigrateDirName)
}

func (s *Store) getSchemaBasePath() string {
	return fmt.Sprintf("migration/%s/%s/", s.profile.Driver, SchemaDirName)
}

func (s *Store) getSeedBasePath() string {
	return fmt.Sprintf("seed/%s/", s.profile.Driver)
}

// seed seeds the database with initial data.
// It reads all seed files from the embedded filesystem and executes them in order.
// This is only supported for SQLite databases and is used in demo mode.
func (s *Store) seed(ctx context.Context) error {
	// Only seed for SQLite - other databases should use production data
	if s.profile.Driver != "sqlite" {
		slog.Warn("seed is only supported for SQLite, skipping for other databases")
		return nil
	}

	filenames, err := fs.Glob(seedFS, fmt.Sprintf("%s*.sql", s.getSeedBasePath()))
	if err != nil {
		return errors.Wrap(err, "failed to read seed files")
	}

	// Sort seed files by name. This is important to ensure that seed files are applied in order.
	sort.Strings(filenames)
	// Start a transaction to apply the seed files.
	tx, err := s.driver.GetDB().Begin()
	if err != nil {
		return errors.Wrap(err, "failed to start transaction")
	}
	defer func() {
		_ = tx.Rollback()
	}()
	// Loop over all seed files and execute them in order.
	for _, filename := range filenames {
		bytes, err := seedFS.ReadFile(filename)
		if err != nil {
			return errors.Wrapf(err, "failed to read seed file, filename=%s", filename)
		}
		if err := s.execute(ctx, tx, string(bytes)); err != nil {
			return errors.Wrapf(err, "seed error: %s", filename)
		}
	}
	return tx.Commit()
}

func (s *Store) GetCurrentSchemaVersion() (string, error) {
	currentVersion := version.GetCurrentVersion(s.profile.Mode)
	// Get all migration files and find the latest one by timestamp
	basePath := s.getMigrationBasePath()
	filePaths, err := fs.Glob(migrationFS, fmt.Sprintf("%s*.up.sql", basePath))
	if err != nil {
		return "", errors.Wrap(err, "failed to read migration files")
	}

	sort.Strings(filePaths)
	if len(filePaths) == 0 {
		return currentVersion, nil
	}
	// Extract timestamp from the latest migration file
	latestFile := filepath.Base(filePaths[len(filePaths)-1])
	parts := strings.SplitN(strings.TrimSuffix(latestFile, ".up.sql"), "_", 2)
	if len(parts) < 1 {
		return currentVersion, nil
	}
	timestamp := parts[0]
	// Return the timestamp as the version (for comparison purposes)
	return timestamp, nil
}

// getSchemaVersionOfMigrateScript extracts the schema version (timestamp) from the migration script file path.
// For new format: returns the 14-digit timestamp (YYYYMMDDHHMMSS).
// For the latest schema file: returns the current schema version.
func (s *Store) getSchemaVersionOfMigrateScript(filePath string) (string, error) {
	// If the file is the latest schema file, return the current schema version.
	if strings.HasSuffix(filePath, LatestSchemaFileName) {
		return s.GetCurrentSchemaVersion()
	}

	// Extract timestamp from filename: YYYYMMDDHHMMSS_description.up.sql
	filename := filepath.Base(filePath)
	// Remove .up.sql extension
	baseName := strings.TrimSuffix(filename, ".up.sql")
	baseName = strings.TrimSuffix(baseName, ".down.sql")
	// Split on first underscore to get timestamp
	parts := strings.SplitN(baseName, "_", 2)
	if len(parts) < 1 {
		return "", errors.Errorf("invalid migration filename: %s", filename)
	}
	timestamp := parts[0]
	// Validate timestamp is 14 digits
	if len(timestamp) != 14 {
		return "", errors.Errorf("migration timestamp must be 14 digits: %s", filename)
	}
	return timestamp, nil
}

// execute executes a SQL statement within a transaction context.
// For PostgreSQL, it splits multi-statement SQL and executes each separately.
// It returns an error if the execution fails.
func (s *Store) execute(ctx context.Context, tx *sql.Tx, stmt string) error {
	// PostgreSQL doesn't support multiple statements in a single ExecContext call.
	// We need to split and execute each statement separately.
	if s.profile.Driver == "postgres" {
		return s.executeMultiStmt(ctx, tx, stmt)
	}
	// For other drivers (SQLite), try single execution first
	if _, err := tx.ExecContext(ctx, stmt); err != nil {
		return errors.Wrap(err, "failed to execute statement")
	}
	return nil
}

// executeMultiStmt splits SQL into individual statements and executes them.
// It handles PostgreSQL's requirement for separate execution of each statement.
func (s *Store) executeMultiStmt(ctx context.Context, tx *sql.Tx, sql string) error {
	statements := s.splitSQL(sql)
	for i, stmt := range statements {
		if stmt == "" {
			continue
		}
		if _, err := tx.ExecContext(ctx, stmt); err != nil {
			return errors.Wrapf(err, "failed to execute statement %d: %s", i+1, stmt)
		}
	}
	return nil
}

// - Preserves function definitions with $$ delimiters.
func (s *Store) splitSQL(sql string) []string {
	var statements []string
	var currentStmt strings.Builder
	lines := strings.Split(sql, "\n")

	inDollarQuote := false
	dollarQuoteTag := ""
	inSingleQuote := false
	inMultiLineComment := false

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Skip pure comment lines
		if strings.HasPrefix(trimmed, "--") && !inDollarQuote && !inSingleQuote && !inMultiLineComment {
			continue
		}

		// Skip empty lines outside of dollar quotes
		if trimmed == "" && !inDollarQuote {
			if currentStmt.Len() > 0 {
				currentStmt.WriteString("\n")
			}
			continue
		}

		// Process each character to track contexts
		i := 0
		for i < len(line) {
			ch := line[i]

			// Check for dollar quote start/end
			if !inSingleQuote && !inMultiLineComment {
				if ch == '$' {
					// Try to match dollar quote tag
					tagEnd := i + 1
					for tagEnd < len(line) && line[tagEnd] != '$' {
						tagEnd++
					}
					if tagEnd < len(line) && line[tagEnd] == '$' {
						tag := line[i : tagEnd+1]
						if inDollarQuote && tag == dollarQuoteTag {
							// Closing dollar quote
							inDollarQuote = false
							dollarQuoteTag = ""
							currentStmt.WriteString(tag)
							i = tagEnd + 1
							continue
						} else if !inDollarQuote {
							// Opening dollar quote
							inDollarQuote = true
							dollarQuoteTag = tag
							currentStmt.WriteString(tag)
							i = tagEnd + 1
							continue
						}
					}
				}
			}

			// Check for single quote
			if ch == '\'' && !inDollarQuote && !inMultiLineComment {
				inSingleQuote = !inSingleQuote
				currentStmt.WriteByte(ch)
				i++
				continue
			}

			// Check for multi-line comment start
			if !inSingleQuote && !inDollarQuote && i+1 < len(line) && line[i:i+2] == "/*" {
				inMultiLineComment = true
				i += 2
				continue
			}

			// Check for multi-line comment end
			if inMultiLineComment && i+1 < len(line) && line[i:i+2] == "*/" {
				inMultiLineComment = false
				i += 2
				continue
			}

			// Skip inline single-line comments
			if !inSingleQuote && !inDollarQuote && !inMultiLineComment && ch == '-' && i+1 < len(line) && line[i+1] == '-' {
				break // Skip rest of line
			}

			// Check for semicolon (statement separator)
			if ch == ';' && !inSingleQuote && !inDollarQuote && !inMultiLineComment {
				currentStmt.WriteByte(ch)
				stmt := strings.TrimSpace(currentStmt.String())
				if stmt != "" {
					statements = append(statements, stmt)
				}
				currentStmt.Reset()
				i++
				// Skip remaining whitespace on this line
				for i < len(line) && (line[i] == ' ' || line[i] == '\t') {
					i++
				}
				continue
			}

			currentStmt.WriteByte(ch)
			i++
		}

		if currentStmt.Len() > 0 {
			currentStmt.WriteString("\n")
		}
	}

	// Add remaining statement (might not end with semicolon)
	if currentStmt.Len() > 0 {
		stmt := strings.TrimSpace(currentStmt.String())
		if stmt != "" {
			statements = append(statements, stmt)
		}
	}

	return statements
}

// updateCurrentSchemaVersion updates the current schema version in the instance basic setting.
// It retrieves the instance basic setting, updates the schema version, and upserts the setting back to the database.
func (s *Store) updateCurrentSchemaVersion(ctx context.Context, schemaVersion string) error {
	instanceBasicSetting, err := s.GetInstanceBasicSetting(ctx)
	if err != nil {
		return errors.Wrap(err, "failed to get instance basic setting")
	}
	instanceBasicSetting.SchemaVersion = schemaVersion
	if _, err := s.UpsertInstanceSetting(ctx, &storepb.InstanceSetting{
		Key:   storepb.InstanceSettingKey_BASIC,
		Value: &storepb.InstanceSetting_BasicSetting{BasicSetting: instanceBasicSetting},
	}); err != nil {
		return errors.Wrap(err, "failed to upsert instance setting")
	}
	return nil
}

// checkMinimumUpgradeVersion verifies the installation meets minimum version requirements for upgrade.
// For very old installations (< v0.22.0), users must upgrade to v0.25.x first before upgrading to current version.
// This is necessary because schema version tracking was moved from migration_history to system_setting in v0.22.0.
func (s *Store) checkMinimumUpgradeVersion(ctx context.Context) error {
	instanceBasicSetting, err := s.GetInstanceBasicSetting(ctx)
	if err != nil {
		return errors.Wrap(err, "failed to get instance basic setting")
	}

	schemaVersion := instanceBasicSetting.SchemaVersion

	// If schema version is >= 0.22.0, the installation is up-to-date
	if !isVersionEmpty(schemaVersion) && version.IsVersionGreaterOrEqualThan(schemaVersion, "0.22.0") {
		return nil
	}

	// If schema version is set but < 0.22.0, this is an old installation
	if !isVersionEmpty(schemaVersion) && !version.IsVersionGreaterOrEqualThan(schemaVersion, "0.22.0") {
		currentVersion, _ := s.GetCurrentSchemaVersion()

		return errors.Errorf(
			"Your Memos installation is too old to upgrade directly.\n\n"+
				"Your current version: %s\n"+
				"Target version: %s\n"+
				"Minimum required: v0.22.0 (May 2024)\n\n"+
				"Upgrade path:\n"+
				"1. First upgrade to v0.25.3: https://github.com/hrygo/divinesense/releases/tag/v0.25.3\n"+
				"2. Start the server and verify it works\n"+
				"3. Then upgrade to the latest version\n\n"+
				"This is required because schema version tracking was moved from migration_history\n"+
				"to system_setting in v0.22.0. The intermediate upgrade handles this migration safely.",
			schemaVersion,
			currentVersion,
		)
	}

	// Schema version is empty - this is either a fresh install or corrupted installation
	// Fresh installs will have schema version set immediately after LATEST.sql is applied
	// So this should not be an issue in normal operation
	return nil
}
