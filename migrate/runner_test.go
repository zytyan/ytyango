package migrate

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
	opts := ExecOptions{DryRun: true, Logf: func(string, ...any) {}}
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
		Logf:       func(string, ...any) {},
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

func TestMainMigrationsCreateGeminiContentV2(t *testing.T) {
	ctx := context.Background()
	dbPath := filepath.Join(t.TempDir(), "main_mig.db")
	db, err := openSQLite(ctx, dbPath)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	// Seed required dependency table for FK.
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
			Migrations:      MigrationsMain,
			ExpectedVersion: ExpectedSchemaVersionMain,
		},
		DBPath: dbPath,
	}
	opts := ExecOptions{Logf: func(string, ...any) {}}
	if err := runCommand(ctx, target, Command{Type: "up"}, opts); err != nil {
		t.Fatalf("apply migrations: %v", err)
	}

	var name string
	if err := db.QueryRowContext(ctx, `SELECT name FROM sqlite_master WHERE type='table' AND name='gemini_content_v2';`).Scan(&name); err != nil {
		t.Fatalf("gemini_content_v2 not found: %v", err)
	}
	if err := db.QueryRowContext(ctx, `SELECT name FROM sqlite_master WHERE type='table' AND name='gemini_content_v2_parts';`).Scan(&name); err != nil {
		t.Fatalf("gemini_content_v2_parts not found: %v", err)
	}

	rows, err := db.QueryContext(ctx, `PRAGMA table_info(gemini_sessions);`)
	if err != nil {
		t.Fatalf("pragma gemini_sessions: %v", err)
	}
	defer rows.Close()
	foundFrozen := false
	foundCacheName := false
	foundCacheTTL := false
	foundCacheExpired := false
	for rows.Next() {
		var cid int
		var cname, ctype string
		var notnull int
		var dflt sql.NullString
		var pk int
		if err := rows.Scan(&cid, &cname, &ctype, &notnull, &dflt, &pk); err != nil {
			t.Fatalf("scan pragma: %v", err)
		}
		switch cname {
		case "frozen":
			foundFrozen = true
		case "cache_name":
			foundCacheName = true
		case "cache_ttl":
			foundCacheTTL = true
		case "cache_expired":
			foundCacheExpired = true
		}
	}
	if foundFrozen {
		t.Fatalf("expected frozen column removed")
	}
	if !(foundCacheName && foundCacheTTL && foundCacheExpired) {
		t.Fatalf("expected cache columns present (cache_name=%v cache_ttl=%v cache_expired=%v)", foundCacheName, foundCacheTTL, foundCacheExpired)
	}

	state, err := ReadState(ctx, db)
	if err != nil {
		t.Fatalf("read state: %v", err)
	}
	if state.Version != ExpectedSchemaVersionMain || state.Dirty {
		t.Fatalf("unexpected state: %+v", state)
	}
}
