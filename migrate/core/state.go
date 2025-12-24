package core

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

const metadataTable = "schema_migrations"

// State represents the current schema version state for one database.
type State struct {
	Version   int
	Dirty     bool
	AppliedAt time.Time
	Log       sql.NullString
}

var errNoState = errors.New("no version state")

// EnsureMetadata creates the schema_migrations table and supporting index if absent.
func EnsureMetadata(ctx context.Context, db *sql.DB) error {
	if db == nil {
		return errors.New("nil db")
	}
	stmts := []string{
		fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %s (
			version INTEGER NOT NULL,
			dirty INTEGER NOT NULL DEFAULT 0,
			applied_at TEXT NOT NULL,
			log TEXT
		);`, metadataTable),
		fmt.Sprintf(`CREATE UNIQUE INDEX IF NOT EXISTS idx_%s_version ON %s(version);`, metadataTable, metadataTable),
	}
	for _, stmt := range stmts {
		if _, err := db.ExecContext(ctx, stmt); err != nil {
			return err
		}
	}
	return nil
}

// ReadState reads the current migration state, creating metadata table if needed.
// If none exists, returns version 0 and dirty=false.
func ReadState(ctx context.Context, db *sql.DB) (State, error) {
	return readState(ctx, db, true)
}

func metadataExists(ctx context.Context, db *sql.DB) (bool, error) {
	row := db.QueryRowContext(ctx, `SELECT 1 FROM sqlite_master WHERE type='table' AND name=?;`, metadataTable)
	var v int
	switch err := row.Scan(&v); err {
	case sql.ErrNoRows:
		return false, nil
	case nil:
		return true, nil
	default:
		return false, err
	}
}

func readState(ctx context.Context, db *sql.DB, ensure bool) (State, error) {
	var st State
	if db == nil {
		return st, errors.New("nil db")
	}
	if ensure {
		if err := EnsureMetadata(ctx, db); err != nil {
			return st, err
		}
	} else {
		exists, err := metadataExists(ctx, db)
		if err != nil {
			return st, err
		}
		if !exists {
			return State{Version: 0, Dirty: false, AppliedAt: time.Time{}}, nil
		}
	}
	row := db.QueryRowContext(ctx, fmt.Sprintf(`SELECT version, dirty, applied_at, log FROM %s LIMIT 1;`, metadataTable))
	var appliedAt string
	var dirtyInt int
	switch err := row.Scan(&st.Version, &dirtyInt, &appliedAt, &st.Log); err {
	case nil:
		st.Dirty = dirtyInt != 0
		t, parseErr := time.Parse(time.RFC3339Nano, appliedAt)
		if parseErr != nil {
			return st, parseErr
		}
		st.AppliedAt = t
		return st, nil
	case sql.ErrNoRows:
		return State{Version: 0, Dirty: false, AppliedAt: time.Time{}}, nil
	default:
		return st, err
	}
}

// writeState overwrites the single state row inside an existing transaction.
func writeState(ctx context.Context, tx *sql.Tx, version int, dirty bool, logText string) error {
	appliedAt := time.Now().UTC().Format(time.RFC3339Nano)
	if _, err := tx.ExecContext(ctx, fmt.Sprintf(`DELETE FROM %s;`, metadataTable)); err != nil {
		return err
	}
	_, err := tx.ExecContext(ctx, fmt.Sprintf(`INSERT INTO %s(version, dirty, applied_at, log) VALUES(?, ?, ?, ?);`, metadataTable),
		version, boolToInt(dirty), appliedAt, logText)
	return err
}

func boolToInt(v bool) int {
	if v {
		return 1
	}
	return 0
}
