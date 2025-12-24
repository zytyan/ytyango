package migrate

import (
	"context"
	"database/sql"
	"fmt"

	_ "github.com/mattn/go-sqlite3"
)

var defaultPragmas = []string{
	"PRAGMA journal_mode=WAL;",
	"PRAGMA wal_autocheckpoint=1000;",
	"PRAGMA synchronous=NORMAL;",
	"PRAGMA mmap_size=67108864; -- 64MB",
	"PRAGMA cache_size=-32768; -- 32MB page cache",
	"PRAGMA busy_timeout=5000;",
	"PRAGMA foreign_keys=ON;",
	"PRAGMA optimize;",
}

// openSQLite opens a SQLite database and applies default pragmas.
func openSQLite(ctx context.Context, path string) (*sql.DB, error) {
	db, err := sql.Open("sqlite3", path)
	if err != nil {
		return nil, err
	}
	for _, pragma := range defaultPragmas {
		if _, execErr := db.ExecContext(ctx, pragma); execErr != nil {
			_ = db.Close()
			return nil, fmt.Errorf("apply pragma %q: %w", pragma, execErr)
		}
	}
	return db, nil
}
