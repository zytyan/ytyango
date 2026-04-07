package handlers

import (
	"context"
	"database/sql"
	"errors"
	g "main/globalcfg"
	"testing"

	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestMeiliWALDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite3", ":memory:")
	require.NoError(t, err)
	require.NoError(t, g.InitMeiliWalDbSchema(db))
	t.Cleanup(func() {
		require.NoError(t, db.Close())
	})
	return db
}

func countMeiliWALRows(t *testing.T, db *sql.DB) int {
	t.Helper()
	var count int
	require.NoError(t, db.QueryRow(`SELECT COUNT(*) FROM meili_wal`).Scan(&count))
	return count
}

func TestInsertMeiliWALStoresJSONContent(t *testing.T) {
	db := newTestMeiliWALDB(t)

	err := insertMeiliWAL(context.Background(), db, `{"mongo_id":"1","peer_id":2,"from_id":3,"msg_id":4,"date":5,"message":"hi"}`)
	require.NoError(t, err)

	var content string
	require.NoError(t, db.QueryRow(`SELECT content FROM meili_wal`).Scan(&content))
	assert.JSONEq(t, `{"mongo_id":"1","peer_id":2,"from_id":3,"msg_id":4,"date":5,"message":"hi"}`, content)
}

func TestFlushMeiliWALDeletesSuccessfulBatches(t *testing.T) {
	db := newTestMeiliWALDB(t)
	require.NoError(t, insertMeiliWAL(context.Background(), db, `{"mongo_id":"1","peer_id":2,"from_id":3,"msg_id":4,"date":5,"message":"first"}`))
	require.NoError(t, insertMeiliWAL(context.Background(), db, `{"mongo_id":"2","peer_id":2,"from_id":3,"msg_id":5,"date":6,"message":"second"}`))
	require.NoError(t, insertMeiliWAL(context.Background(), db, `{"mongo_id":"3","peer_id":2,"from_id":3,"msg_id":6,"date":7,"message":"third"}`))

	var batchSizes []int
	err := flushMeiliWAL(context.Background(), db, 2, func(data any) error {
		docs, ok := data.([]MeiliMsg)
		require.True(t, ok)
		batchSizes = append(batchSizes, len(docs))
		return nil
	})
	require.NoError(t, err)

	assert.Equal(t, []int{2, 1}, batchSizes)
	assert.Equal(t, 0, countMeiliWALRows(t, db))
}

func TestFlushMeiliWALRollsBackFailedBatch(t *testing.T) {
	db := newTestMeiliWALDB(t)
	require.NoError(t, insertMeiliWAL(context.Background(), db, `{"mongo_id":"1","peer_id":2,"from_id":3,"msg_id":4,"date":5,"message":"first"}`))
	require.NoError(t, insertMeiliWAL(context.Background(), db, `{"mongo_id":"2","peer_id":2,"from_id":3,"msg_id":5,"date":6,"message":"second"}`))

	err := flushMeiliWAL(context.Background(), db, 500, func(data any) error {
		return errors.New("meili down")
	})
	require.Error(t, err)

	assert.Equal(t, 2, countMeiliWALRows(t, db))
}

func TestFlushMeiliWALKeepsOnlyFailedLaterBatch(t *testing.T) {
	db := newTestMeiliWALDB(t)
	require.NoError(t, insertMeiliWAL(context.Background(), db, `{"mongo_id":"1","peer_id":2,"from_id":3,"msg_id":4,"date":5,"message":"first"}`))
	require.NoError(t, insertMeiliWAL(context.Background(), db, `{"mongo_id":"2","peer_id":2,"from_id":3,"msg_id":5,"date":6,"message":"second"}`))
	require.NoError(t, insertMeiliWAL(context.Background(), db, `{"mongo_id":"3","peer_id":2,"from_id":3,"msg_id":6,"date":7,"message":"third"}`))

	calls := 0
	err := flushMeiliWAL(context.Background(), db, 2, func(data any) error {
		calls++
		if calls == 2 {
			return errors.New("meili down")
		}
		return nil
	})
	require.Error(t, err)

	assert.Equal(t, 1, countMeiliWALRows(t, db))
}
