package meilisearch

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAddDocumentsPostsJSONBatch(t *testing.T) {
	var gotAuth string
	var gotContentType string
	var gotPath string
	var gotPrimaryKey string
	var gotBody []map[string]any

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		gotContentType = r.Header.Get("Content-Type")
		gotPath = r.URL.Path
		gotPrimaryKey = r.URL.Query().Get("primaryKey")
		require.NoError(t, json.NewDecoder(r.Body).Decode(&gotBody))
		w.WriteHeader(http.StatusAccepted)
	}))
	t.Cleanup(server.Close)

	client := NewMeiliClient(server.URL, "tgmsgs", "secret", "mongo_id")
	err := client.AddDocuments([]map[string]any{
		{"mongo_id": "1", "message": "first"},
		{"mongo_id": "2", "message": "second"},
	})
	require.NoError(t, err)

	assert.Equal(t, "Bearer secret", gotAuth)
	assert.Equal(t, "application/json", gotContentType)
	assert.Equal(t, "/indexes/tgmsgs/documents", gotPath)
	assert.Equal(t, "mongo_id", gotPrimaryKey)
	assert.Len(t, gotBody, 2)
	assert.Equal(t, "first", gotBody[0]["message"])
	assert.Equal(t, "second", gotBody[1]["message"])
}
