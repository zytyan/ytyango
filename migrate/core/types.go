package core

import (
	"context"
	"database/sql"
	"errors"
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

type Registry struct {
	Name            string
	Migrations      []Migration
	ExpectedVersion int
}

// Target bundles a registry with its concrete DB path.
type Target struct {
	Registry Registry
	DBPath   string
}

type ExecOptions struct {
	DryRun bool
	Logf   func(string, ...any)
}

type Command struct {
	Type string // up, down, to, status
	To   int
	Step int
}

var ErrDirtyDatabase = errors.New("database is marked dirty")
