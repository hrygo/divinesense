//go:build !sqlite_vec && cgo
// +build !sqlite_vec,cgo

package sqlite

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/mattn/go-sqlite3"
	_ "github.com/mattn/go-sqlite3"
)

// loadExtension loads a SQLite extension using go-sqlite3's built-in LoadExtension method.
// go-sqlite3 includes extension loading support by default (build tag: !sqlite_omit_load_extension).
func loadExtension(db *sql.DB, extensionPath string) error {
	// Get the underlying driver connection
	conn, err := db.Conn(context.Background())
	if err != nil {
		return fmt.Errorf("failed to get connection: %w", err)
	}
	defer conn.Close()

	// Use Raw() to access the underlying sqlite3 connection
	err = conn.Raw(func(driverConn interface{}) error {
		// The driver connection should be a *sqlite3.SQLiteConn
		sqliteConn, ok := driverConn.(*sqlite3.SQLiteConn)
		if !ok {
			return fmt.Errorf("unexpected driver connection type: %T", driverConn)
		}

		// Load the extension with the correct entry point
		// sqlite-vec uses "sqlite3_vec_init" as the entry point
		return sqliteConn.LoadExtension(extensionPath, "sqlite3_vec_init")
	})

	return err
}
