//go:build !sqlite_vec && !cgo
// +build !sqlite_vec,!cgo

package sqlite

import (
	"database/sql"

	"github.com/pkg/errors"
)

// loadVecExtension is a stub for non-CGO environments (e.g., static builds).
// In non-CGO environments, we cannot load SQLite extensions, so vector search
// will always use the Go-based cosine similarity fallback.
func loadVecExtension(db *sql.DB) error {
	// Return an error to indicate extension is not available
	// The caller will log a warning and set vecExtensionLoaded = false
	return errors.New("sqlite-vec extension not available in non-CGO builds")
}
