package migrate

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"

	"go.uber.org/zap"
)

func TestEnsureMetadataAndCheckVersion(t *testing.T) {
	ctx := context.Background()
	dbPath := filepath.Join(t.TempDir(), "meta.db")
	db, err := openSQLite(ctx, dbPath)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer db.Close()

	st, err := ReadState(ctx, db)
	if err != nil {
		t.Fatalf("read state: %v", err)
	}
	if st.Version != 0 || st.Dirty {
		t.Fatalf("unexpected initial state: %+v", st)
	}
	if err := CheckVersion(ctx, db, 0, "main"); err != nil {
		t.Fatalf("check version: %v", err)
	}
}

func TestDryRunDoesNotMutate(t *testing.T) {
	ctx := context.Background()
	dbPath := filepath.Join(t.TempDir(), "dry.db")
	db, err := openSQLite(ctx, dbPath)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	db.Close()

	migs := []Migration{
		{
			Version: 1,
			Name:    "create demo",
			Up: []Step{
				{SQL: []string{"CREATE TABLE demo(id INTEGER);"}},
			},
			Down: []Step{
				{SQL: []string{"DROP TABLE demo;"}},
			},
		},
	}
	target := Target{
		Registry: Registry{Name: "main", Migrations: migs, ExpectedVersion: 1},
		DBPath:   dbPath,
	}
	opts := ExecOptions{DryRun: true, Logger: zap.NewNop().Sugar()}
	if err := runCommand(ctx, target, Command{Type: "up"}, opts); err != nil {
		t.Fatalf("dry-run migrate: %v", err)
	}

	db, err = openSQLite(ctx, dbPath)
	if err != nil {
		t.Fatalf("reopen db: %v", err)
	}
	defer db.Close()

	var name string
	err = db.QueryRowContext(ctx, `SELECT name FROM sqlite_master WHERE type='table' AND name='demo';`).Scan(&name)
	if err == nil {
		t.Fatalf("table should not exist on dry-run, found %s", name)
	}
	if err != sql.ErrNoRows {
		t.Fatalf("unexpected error checking table: %v", err)
	}
	state, err := ReadState(ctx, db)
	if err != nil {
		t.Fatalf("read state: %v", err)
	}
	if state.Version != 0 || state.Dirty {
		t.Fatalf("expected state unchanged, got %+v", state)
	}
}

func TestMemoryRunSamplingDoesNotTouchDisk(t *testing.T) {
	ctx := context.Background()
	dbPath := filepath.Join(t.TempDir(), "memrun.db")
	db, err := openSQLite(ctx, dbPath)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if _, err := db.ExecContext(ctx, `CREATE TABLE items(id INTEGER PRIMARY KEY, val TEXT);`); err != nil {
		t.Fatalf("create table: %v", err)
	}
	for i := 0; i < 50; i++ {
		if _, err := db.ExecContext(ctx, `INSERT INTO items(val) VALUES (?);`, i); err != nil {
			t.Fatalf("insert: %v", err)
		}
	}
	db.Close()

	migs := []Migration{
		{
			Version: 1,
			Name:    "add note",
			Up: []Step{
				{SQL: []string{"ALTER TABLE items ADD COLUMN note TEXT;"}},
			},
			Down: []Step{
				{SQL: []string{"CREATE TABLE items_tmp(id INTEGER PRIMARY KEY, val TEXT); INSERT INTO items_tmp(id,val) SELECT id,val FROM items; DROP TABLE items; ALTER TABLE items_tmp RENAME TO items;"}},
			},
		},
	}
	target := Target{
		Registry: Registry{Name: "main", Migrations: migs, ExpectedVersion: 1},
		DBPath:   dbPath,
	}
	opts := ExecOptions{
		MemoryRun:  true,
		SampleRate: 0.2,
		Logger:     zap.NewNop().Sugar(),
	}
	if err := runCommand(ctx, target, Command{Type: "up"}, opts); err != nil {
		t.Fatalf("memory-run migrate: %v", err)
	}

	db, err = openSQLite(ctx, dbPath)
	if err != nil {
		t.Fatalf("reopen db: %v", err)
	}
	defer db.Close()
	rows, err := db.QueryContext(ctx, `PRAGMA table_info(items);`)
	if err != nil {
		t.Fatalf("pragma: %v", err)
	}
	defer rows.Close()
	cols := 0
	for rows.Next() {
		cols++
	}
	if cols != 2 {
		t.Fatalf("expected original table with 2 columns after memory-run, got %d", cols)
	}
}
