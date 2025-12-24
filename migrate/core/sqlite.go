package core

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	_ "github.com/mattn/go-sqlite3"
)

var (
	defaultPragmas = []string{
		"PRAGMA foreign_keys=ON;",
	}
	readonlyPragmas = []string{
		"PRAGMA foreign_keys=ON;",
	}
)

// openSQLite opens a SQLite database and applies pragmas.
func openSQLite(ctx context.Context, path string, readonly bool) (*sql.DB, error) {
	dsn := path
	if readonly {
		if strings.HasPrefix(path, "file:") {
			dsn = path + "&mode=ro"
		} else {
			dsn = fmt.Sprintf("file:%s?mode=ro", path)
		}
	}
	db, err := sql.Open("sqlite3", dsn)
	if err != nil {
		return nil, err
	}
	pragmas := defaultPragmas
	if readonly {
		pragmas = readonlyPragmas
	}
	for _, pragma := range pragmas {
		if _, execErr := db.ExecContext(ctx, pragma); execErr != nil {
			_ = db.Close()
			return nil, fmt.Errorf("apply pragma %q: %w", pragma, execErr)
		}
	}
	return db, nil
}
