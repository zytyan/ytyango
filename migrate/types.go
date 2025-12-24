package migrate

import (
	"context"
	"database/sql"
)

// Step represents one executable unit in a migration.
// SQL is executed in order; Go allows custom logic. Either may be empty.
type Step struct {
	Description string
	SQL         []string
	Go          func(ctx context.Context, tx *sql.Tx) error
}

// Migration is a single versioned change set.
type Migration struct {
	Version int
	Name    string
	Up      []Step
	Down    []Step
}
