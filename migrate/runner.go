package migrate

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"
)

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

var defaultLogf = func(format string, args ...any) {
	fmt.Printf(format, args...)
}

func registrySet() map[string]Registry {
	return map[string]Registry{
		"main": {Name: "main", Migrations: MigrationsMain, ExpectedVersion: ExpectedSchemaVersionMain},
		"msg":  {Name: "msg", Migrations: MigrationsMsg, ExpectedVersion: ExpectedSchemaVersionMsg},
	}
}

func latestVersion(migs []Migration) int {
	max := 0
	for _, m := range migs {
		if m.Version > max {
			max = m.Version
		}
	}
	return max
}

func validateMigrations(migs []Migration) error {
	if len(migs) == 0 {
		return nil
	}
	seen := make(map[int]struct{}, len(migs))
	for _, m := range migs {
		if _, ok := seen[m.Version]; ok {
			return fmt.Errorf("duplicate migration version %d", m.Version)
		}
		seen[m.Version] = struct{}{}
	}
	versions := make([]int, 0, len(migs))
	for v := range seen {
		versions = append(versions, v)
	}
	sort.Ints(versions)
	for i, v := range versions {
		expected := i + 1
		if v != expected {
			return fmt.Errorf("migration versions must be contiguous starting at 1; saw %d expected %d", v, expected)
		}
	}
	return nil
}

func findMigration(migs []Migration, version int) (Migration, bool) {
	for _, m := range migs {
		if m.Version == version {
			return m, true
		}
	}
	return Migration{}, false
}

func runCommand(ctx context.Context, target Target, cmd Command, opts ExecOptions) error {
	reg := target.Registry
	if reg.Name == "" {
		return errors.New("registry missing name")
	}
	if opts.Logf == nil {
		opts.Logf = defaultLogf
	}
	if err := validateMigrations(reg.Migrations); err != nil {
		return fmt.Errorf("validate migrations for %s: %w", reg.Name, err)
	}
	if target.DBPath == "" {
		return fmt.Errorf("%s db path is empty", reg.Name)
	}
	db, cleanup, err := openForRun(ctx, target.DBPath, opts)
	if err != nil {
		return err
	}
	defer cleanup()

	ensureMeta := !(opts.DryRun || cmd.Type == "status")
	state, err := readState(ctx, db, ensureMeta)
	if err != nil {
		return fmt.Errorf("read %s state: %w", reg.Name, err)
	}
	if state.Dirty && cmd.Type != "status" {
		return fmt.Errorf("%w for %s (version=%d)", ErrDirtyDatabase, reg.Name, state.Version)
	}

	switch cmd.Type {
	case "status":
		printState(reg.Name, target.DBPath, state, reg.ExpectedVersion, opts.Logf)
		return nil
	case "up":
		target := cmd.To
		if target == 0 {
			target = latestVersion(reg.Migrations)
		}
		if target < state.Version {
			return fmt.Errorf("%s already at version %d (target %d lower); use down/to", reg.Name, state.Version, target)
		}
		return applyRange(ctx, db, reg, state.Version, target, true, opts)
	case "down":
		target := cmd.To
		if target < 0 {
			step := cmd.Step
			if step <= 0 {
				step = 1
			}
			target = state.Version - step
		}
		if target < 0 {
			target = 0
		}
		return applyRange(ctx, db, reg, state.Version, target, false, opts)
	case "to":
		return applyRange(ctx, db, reg, state.Version, cmd.To, cmd.To >= state.Version, opts)
	default:
		return fmt.Errorf("unknown command %q", cmd.Type)
	}
}

func applyRange(ctx context.Context, db *sql.DB, reg Registry, current, target int, ascending bool, opts ExecOptions) error {
	if current == target {
		if opts.Logf != nil {
			opts.Logf("[%s] already at version %d\n", reg.Name, current)
		}
		return nil
	}
	if ascending {
		for v := current + 1; v <= target; v++ {
			mig, ok := findMigration(reg.Migrations, v)
			if !ok {
				return fmt.Errorf("[%s] missing migration %d", reg.Name, v)
			}
			if err := applyMigration(ctx, db, reg.Name, mig, true, opts); err != nil {
				return err
			}
		}
		return nil
	}
	// descending
	for v := current; v > target; v-- {
		mig, ok := findMigration(reg.Migrations, v)
		if !ok {
			return fmt.Errorf("[%s] missing migration %d for down", reg.Name, v)
		}
		if err := applyMigration(ctx, db, reg.Name, mig, false, opts); err != nil {
			return err
		}
	}
	return nil
}

func applyMigration(ctx context.Context, db *sql.DB, name string, mig Migration, up bool, opts ExecOptions) error {
	dir := "up"
	targetVersion := mig.Version
	steps := mig.Up
	if !up {
		dir = "down"
		targetVersion = mig.Version - 1
		steps = mig.Down
	}

	if targetVersion < 0 {
		targetVersion = 0
	}

	if len(steps) == 0 {
		return fmt.Errorf("[%s] migration %d (%s) has no %s steps", name, mig.Version, mig.Name, dir)
	}

	if opts.Logf != nil {
		opts.Logf("[%s] %s migration %d (%s) -> version %d\n", name, strings.ToUpper(dir), mig.Version, mig.Name, targetVersion)
	}

	if opts.DryRun {
		for _, step := range steps {
			for _, stmt := range step.SQL {
				if opts.Logf != nil {
					opts.Logf("[dry-run][%s] SQL: %s\n", name, strings.TrimSpace(stmt))
				}
			}
			if step.Go != nil && opts.Logf != nil {
				opts.Logf("[dry-run][%s] Go step: %s\n", name, step.Description)
			}
		}
		return nil
	}

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("[%s] begin tx: %w", name, err)
	}
	if err := writeState(ctx, tx, targetVersion, true, fmt.Sprintf("applying %s %d (%s)", dir, mig.Version, mig.Name)); err != nil {
		_ = tx.Rollback()
		return fmt.Errorf("[%s] mark dirty: %w", name, err)
	}
	for _, step := range steps {
		for _, stmt := range step.SQL {
			if opts.Logf != nil {
				opts.Logf("[%s] exec SQL: %s\n", name, strings.TrimSpace(stmt))
			}
			if _, err := tx.ExecContext(ctx, stmt); err != nil {
				_ = tx.Rollback()
				return fmt.Errorf("[%s] migration %d (%s) %s step failed: %w", name, mig.Version, mig.Name, dir, err)
			}
		}
		if step.Go != nil {
			if opts.Logf != nil && step.Description != "" {
				opts.Logf("[%s] Go step: %s\n", name, step.Description)
			}
			if err := step.Go(ctx, tx); err != nil {
				_ = tx.Rollback()
				return fmt.Errorf("[%s] migration %d (%s) %s Go step failed: %w", name, mig.Version, mig.Name, dir, err)
			}
		}
	}
	if err := writeState(ctx, tx, targetVersion, false, fmt.Sprintf("applied %s %d (%s)", dir, mig.Version, mig.Name)); err != nil {
		_ = tx.Rollback()
		return fmt.Errorf("[%s] set version: %w", name, err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("[%s] commit: %w", name, err)
	}
	return nil
}

func openForRun(ctx context.Context, path string, opts ExecOptions) (*sql.DB, func(), error) {
	db, err := openSQLite(ctx, path, opts.DryRun)
	if err != nil {
		return nil, func() {}, err
	}
	return db, func() { _ = db.Close() }, nil
}

func printState(name, path string, st State, expected int, logf func(string, ...any)) {
	if logf == nil {
		return
	}
	logf("[%s] path=%s version=%d dirty=%v expected=%d applied_at=%s\n", name, path, st.Version, st.Dirty, expected, st.AppliedAt.Format(time.RFC3339Nano))
}
