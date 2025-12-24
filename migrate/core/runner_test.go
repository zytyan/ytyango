package core

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"testing"
)

func TestEnsureMetadataAndCheckVersion(t *testing.T) {
	ctx := context.Background()
	dbPath := filepath.Join(t.TempDir(), "meta.db")
	db, err := openSQLite(ctx, dbPath, false)
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
	db, err := openSQLite(ctx, dbPath, false)
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
	opts := ExecOptions{DryRun: true, Logf: func(string, ...any) {}}
	if err := RunCommand(ctx, target, Command{Type: "up"}, opts); err != nil {
		t.Fatalf("dry-run migrate: %v", err)
	}

	db, err = openSQLite(ctx, dbPath, false)
	if err != nil {
		t.Fatalf("reopen db: %v", err)
	}
	defer db.Close()

	var name string
	err = db.QueryRowContext(ctx, `SELECT name FROM sqlite_master WHERE type='table' AND name='demo';`).Scan(&name)
	if err == nil {
		t.Fatalf("table should not exist on dry-run, found %s", name)
	}
	if !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("unexpected error checking table: %v", err)
	}
	state, err := readState(ctx, db, false)
	if err != nil {
		t.Fatalf("read state: %v", err)
	}
	if state.Version != 0 || state.Dirty {
		t.Fatalf("expected state unchanged, got %+v", state)
	}
	var metaName string
	if err := db.QueryRowContext(ctx, `SELECT name FROM sqlite_master WHERE type='table' AND name='schema_migrations';`).Scan(&metaName); err == nil {
		t.Fatalf("expected no schema_migrations table on dry-run")
	} else if !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("unexpected error checking schema_migrations: %v", err)
	}
}

func TestMainMigrationsCreateGeminiContentV2(t *testing.T) {
	ctx := context.Background()
	dbPath := filepath.Join(t.TempDir(), "main_mig.db")
	db, err := openSQLite(ctx, dbPath, false)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	_, err = db.ExecContext(ctx, `CREATE TABLE gemini_sessions(
id INTEGER PRIMARY KEY AUTOINCREMENT,
chat_id INTEGER NOT NULL,
chat_name TEXT NOT NULL,
chat_type TEXT NOT NULL,
frozen INTEGER NOT NULL DEFAULT 0
) STRICT;`)
	if err != nil {
		t.Fatalf("create gemini_sessions: %v", err)
	}

	target := Target{
		Registry: Registry{
			Name:            "main",
			Migrations:      []Migration{},
			ExpectedVersion: 0,
		},
		DBPath: dbPath,
	}
	opts := ExecOptions{Logf: func(string, ...any) {}}
	if err := RunCommand(ctx, target, Command{Type: "up"}, opts); err != nil {
		t.Fatalf("apply migrations: %v", err)
	}

	var name string
	if err := db.QueryRowContext(ctx, `SELECT name FROM sqlite_master WHERE type='table' AND name='gemini_content_v2';`).Scan(&name); err == nil {
		t.Fatalf("unexpected gemini_content_v2 should not exist without registry migrations")
	}
}
