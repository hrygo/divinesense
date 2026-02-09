//go:build sqlite_vec
// +build sqlite_vec

//go:generate ./download_sqlite_vec.sh

package sqlite

/*
#cgo CFLAGS: -I${SRCDIR}/.lib
#cgo LDFLAGS: ${SRCDIR}/.lib/libvec0.a

#include <sqlite3.h>

// Declare the sqlite3_vec_init function from the static library
int sqlite3_vec_init(sqlite3 *db, char **pzErrMsg, const sqlite3_api_routines *pApi);
*/
import "C"

import (
	"database/sql"
	"log/slog"

	"github.com/pkg/errors"
)

// Auto registers the sqlite-vec extension as an auto-extension.
// This must be called before opening any database connections.
//
// This function uses sqlite3_auto_extension() to register sqlite3_vec_init,
// which will be automatically called for each new database connection.
func Auto() error {
	// Register the extension using sqlite3_auto_extension
	// This tells SQLite to automatically call sqlite3_vec_init for each new DB connection
	rc := C.sqlite3_auto_extension((*[0]byte)(C.sqlite3_vec_init))
	if rc != 0 {
		return errors.New("failed to register sqlite-vec auto-extension")
	}
	slog.Info("sqlite-vec extension registered as auto-extension")
	return nil
}

func init() {
	// Automatically register the extension when this package is imported
	if err := Auto(); err != nil {
		slog.Error("Failed to auto-register sqlite-vec extension", "error", err)
	}
}

// loadVecExtension verifies the sqlite-vec extension is loaded from static library.
// The statically linked libvec0.a should auto-register via sqlite3_auto_extension.
func loadVecExtension(db *sql.DB) error {
	// Verify the extension is working by checking if vec0 functions are available
	var result int
	err := db.QueryRow("SELECT count(*) FROM pragma_function_list WHERE name LIKE 'vec_%'").Scan(&result)
	if err != nil {
		return errors.Wrap(err, "failed to verify sqlite-vec extension")
	}

	if result == 0 {
		slog.Warn("sqlite-vec extension not loaded, vector search will use Go fallback")
		return errors.New("sqlite-vec extension not loaded (no vec_ functions found)")
	}

	slog.Info("sqlite-vec extension verified (static linking)", "functions_found", result)
	return nil
}
