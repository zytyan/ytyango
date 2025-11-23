package q

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"testing"

	_ "github.com/mattn/go-sqlite3"
)

func setupPicsTest(t *testing.T) (*sql.DB, *Queries) {
	t.Helper()

	// reset shared prefix-sum state between tests
	psMu.Lock()
	countByRatePrefixSum = nil
	minCountRate = 0
	psMu.Unlock()

	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}

	schemaPath := filepath.Join("..", "..", "sql", "schema_pics.sql")
	schema, err := os.ReadFile(schemaPath)
	if err != nil {
		t.Fatalf("read schema: %v", err)
	}
	if _, err = db.Exec(string(schema)); err != nil {
		t.Fatalf("init schema: %v", err)
	}

	q := New(db)

	t.Cleanup(func() {
		_ = db.Close()
	})
	return db, q
}

func addTestPic(t *testing.T, db *sql.DB, uid, fid string, rate, randKey int64) {
	t.Helper()
	_, err := db.Exec(`INSERT INTO saved_pics (file_uid, file_id, bot_rate, rand_key, user_rate) VALUES (?, ?, ?, ?, ?)`,
		uid, fid, rate, randKey, rate)
	if err != nil {
		t.Fatalf("insert pic %s: %v", uid, err)
	}
}

func TestGetPicByUserRateRangeLazyInit(t *testing.T) {
	db, q := setupPicsTest(t)
	addTestPic(t, db, "u1", "f1", 2, 10)
	addTestPic(t, db, "u2", "f2", 4, 20)
	got, err := q.GetPicByUserRateRange(context.Background(), 0, 6)
	if err != nil {
		t.Fatalf("GetPicByUserRateRange returned error: %v", err)
	}
	if got == "" {
		t.Fatalf("expected a file id, got empty string")
	}

	psMu.RLock()
	defer psMu.RUnlock()
	if countByRatePrefixSum == nil {
		t.Fatalf("prefix sum not initialized after first call")
	}
}

func TestGetPicByUserRateRangeRespectsBounds(t *testing.T) {
	db, q := setupPicsTest(t)
	addTestPic(t, db, "u1", "f1", 2, 10)
	addTestPic(t, db, "u2", "f2", 4, 20)
	addTestPic(t, db, "u3", "f3", 6, 30)

	id, err := q.GetPicByUserRateRange(context.Background(), 0, 4)
	if err != nil {
		t.Fatalf("GetPicByUserRateRange(0,4) error: %v", err)
	}

	var pickedRate int
	if err := db.QueryRow(`SELECT user_rate FROM saved_pics WHERE file_id = ?`, id).Scan(&pickedRate); err != nil {
		t.Fatalf("lookup picked rate: %v", err)
	}
	if pickedRate < 0 || pickedRate > 4 {
		t.Fatalf("picked rate %d outside expected range [0,4]", pickedRate)
	}

	_, err = q.GetPicByUserRateRange(context.Background(), -5, -1)
	if !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("expected sql.ErrNoRows for empty intersection, got %v", err)
	}
}
