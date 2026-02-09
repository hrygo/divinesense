//go:build sqlite_vec
// +build sqlite_vec

//go:generate ./download_sqlite_vec.sh

package sqlite

/*
#cgo CFLAGS: -I${SRCDIR}/.lib
#cgo LDFLAGS: ${SRCDIR}/.lib/libvec0.a

#include <sqlite3.h>
*/
import "C"

import (
	"database/sql"
	"log/slog"

	"github.com/pkg/errors"
)

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
