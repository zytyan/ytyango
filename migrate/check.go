package migrate

import (
	"context"
	"database/sql"
	"fmt"
)

// CheckVersion enforces that the DB schema version matches the expected value and is not dirty.
func CheckVersion(ctx context.Context, db *sql.DB, expected int, name string) error {
	state, err := ReadState(ctx, db)
	if err != nil {
		return fmt.Errorf("read %s schema state: %w", name, err)
	}
	if state.Dirty {
		return fmt.Errorf("%s schema is dirty at version %d; run migrate", name, state.Version)
	}
	if state.Version != expected {
		return fmt.Errorf("%s schema version mismatch: current=%d expected=%d; run migrate", name, state.Version, expected)
	}
	return nil
}
