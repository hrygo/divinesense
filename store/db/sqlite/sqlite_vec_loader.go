//go:build !sqlite_vec
// +build !sqlite_vec

package sqlite

import (
	"database/sql"
	"log/slog"

	"github.com/pkg/errors"
)

// loadVecExtension loads the sqlite-vec extension for vector search operations.
// This enables the vec0 virtual table and distance functions for efficient similarity search.
func loadVecExtension(db *sql.DB) error {
	// Try to load the extension from common locations
	// Priority: local build -> system install -> standard paths
	extensionPaths := []string{
		// Local development path (auto-downloaded via go generate)
		"./store/db/sqlite/.lib/libvec0.dylib",
		// System-wide installation (via pkg install, apt, etc.)
		"vec0",
		// Standard library paths
		"/usr/local/lib/libvec0.dylib",
		"/opt/homebrew/lib/libvec0.dylib",
		"/usr/lib/libvec0.so",
		"/usr/lib/x86_64-linux-gnu/libvec0.so",
	}

	var lastErr error
	var loadedPath string
	for i, path := range extensionPaths {
		slog.Debug("Attempting to load sqlite-vec extension", "attempt", i+1, "total", len(extensionPaths), "path", path)
		if err := loadExtension(db, path); err == nil {
			loadedPath = path
			slog.Info("sqlite-vec extension loaded successfully", "path", path)
			break
		} else {
			slog.Warn("sqlite-vec extension load failed", "attempt", i+1, "path", path, "error", err)
			lastErr = err
			// Continue trying next path
		}
	}

	if loadedPath == "" {
		slog.Error("Failed to load sqlite-vec extension from all locations",
			"attempted_count", len(extensionPaths),
			"last_error", lastErr)
		return errors.Wrapf(lastErr, "failed to load sqlite-vec from any location (tried %d paths)", len(extensionPaths))
	}

	slog.Info("sqlite-vec extension loaded and verified",
		"path", loadedPath,
	)

	return nil
}
