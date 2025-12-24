package handlers

import (
	"archive/tar"
	"bytes"
	"encoding/json"
	"io"
	g "main/globalcfg"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/klauspost/compress/zstd"
	"go.uber.org/zap"
)

func newTestServer() *httptest.Server {
	logger := g.GetLogger("inner-http-test", zap.InfoLevel)
	return httptest.NewServer(buildHandler(logger.Desugar()))
}

func tarZstdEntries(t *testing.T, data []byte) map[string][]byte {
	t.Helper()
	decoder, err := zstd.NewReader(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("open zstd: %v", err)
	}
	defer decoder.Close()
	tarReader := tar.NewReader(decoder)
	entries := make(map[string][]byte)
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("iterate tar: %v", err)
		}
		content, err := io.ReadAll(tarReader)
		if err != nil {
			t.Fatalf("read tar file %s: %v", header.Name, err)
		}
		entries[header.Name] = content
	}
	return entries
}

func parseManifest(t *testing.T, data []byte) backupManifest {
	t.Helper()
	var manifest backupManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		t.Fatalf("unmarshal manifest: %v", err)
	}
	return manifest
}

func TestMarsCounterSuccess(t *testing.T) {
	server := newTestServer()
	defer server.Close()

	groupID := time.Now().UnixNano()
	resp, err := http.Post(server.URL+"/mars-counter", "application/json", strings.NewReader(`{"group_id":`+strconv.FormatInt(groupID, 10)+`,"mars_count":2}`))
	if err != nil {
		t.Fatalf("post mars-counter: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("unexpected status: %d", resp.StatusCode)
	}

	stat := g.Q.ChatStatNow(groupID)
	if stat == nil {
		t.Fatalf("Q.ChatStatNow(groupID) is nil")
	}
	if stat.MarsCount != 1 {
		t.Fatalf("expected MarsCount=1 got %d", stat.MarsCount)
	}
	if stat.MaxMarsCount != 2 {
		t.Fatalf("expected MaxMarsCount=2 got %d", stat.MaxMarsCount)
	}
}

func TestMarsCounterBadJSON(t *testing.T) {
	server := newTestServer()
	defer server.Close()

	resp, err := http.Post(server.URL+"/mars-counter", "application/json", strings.NewReader("{invalid"))
	if err != nil {
		t.Fatalf("post mars-counter: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 got %d", resp.StatusCode)
	}
}

func TestDioBanActions(t *testing.T) {
	server := newTestServer()
	defer server.Close()

	groupID := time.Now().UnixNano()
	body := `{"user_id":1,"group_id":` + strconv.FormatInt(groupID, 10) + `,"action":0}`
	resp, err := http.Post(server.URL+"/dio-ban", "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatalf("post dio-ban add: %v", err)
	}
	resp.Body.Close()

	body = `{"user_id":1,"group_id":` + strconv.FormatInt(groupID, 10) + `,"action":2}`
	resp, err = http.Post(server.URL+"/dio-ban", "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatalf("post dio-ban ban: %v", err)
	}
	resp.Body.Close()

	stat := g.Q.ChatStatNow(groupID)
	if stat == nil {
		t.Fatalf("expected stat != nil, got nil")
	}
	if stat.DioAddUserCount != 1 {
		t.Fatalf("expected DioAddUserCount=1 got %d", stat.DioAddUserCount)
	}
	if stat.DioBanUserCount != 1 {
		t.Fatalf("expected DioBanUserCount=1 got %d", stat.DioBanUserCount)
	}
}

func TestSetLoggerLevelNotFound(t *testing.T) {
	server := newTestServer()
	defer server.Close()

	req, err := http.NewRequest(http.MethodPut, server.URL+"/loggers/not-exist/1", nil)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("do request: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 got %d", resp.StatusCode)
	}
	body, _ := io.ReadAll(resp.Body)
	if !strings.Contains(string(body), "logger not-exist not found") {
		t.Fatalf("unexpected body: %s", string(body))
	}
}

func TestPprofIndexAvailable(t *testing.T) {
	server := newTestServer()
	defer server.Close()

	resp, err := http.Get(server.URL + "/debug/pprof/")
	if err != nil {
		t.Fatalf("get pprof: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 got %d", resp.StatusCode)
	}
}

func TestResolveInnerHTTPAddr(t *testing.T) {
	addr, enabled := resolveInnerHTTPAddr("")
	if !enabled || addr != innerHTTPDefaultAddr {
		t.Fatalf("expected default addr, got %s enabled=%v", addr, enabled)
	}

	addr, enabled = resolveInnerHTTPAddr("OFF")
	if enabled || addr != "" {
		t.Fatalf("expected disabled on OFF, got addr=%s enabled=%v", addr, enabled)
	}

	addr, enabled = resolveInnerHTTPAddr("127.0.0.1:12345")
	if !enabled || addr != "127.0.0.1:12345" {
		t.Fatalf("unexpected addr parsing, got %s enabled=%v", addr, enabled)
	}
}

func TestResolveInnerHTTPAddrPanicsOnInvalid(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatalf("expected panic for invalid addr")
		}
	}()
	resolveInnerHTTPAddr("invalid-addr")
}

func TestRootListsRoutesForAnyMethod(t *testing.T) {
	server := newTestServer()
	defer server.Close()

	req, err := http.NewRequest(http.MethodPost, server.URL+"/", nil)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("do request: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 got %d", resp.StatusCode)
	}
	body, _ := io.ReadAll(resp.Body)
	if !strings.Contains(string(body), "GET /loggers") {
		t.Fatalf("unexpected body: %s", string(body))
	}
}

func TestBackupDBSuccessAll(t *testing.T) {
	server := newTestServer()
	defer server.Close()

	resp, err := http.Get(server.URL + "/backupdb")
	if err != nil {
		t.Fatalf("get backupdb: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 got %d", resp.StatusCode)
	}
	if ct := resp.Header.Get("Content-Type"); ct != "application/zstd" {
		t.Fatalf("expected content-type application/zstd, got %s", ct)
	}
	if cd := resp.Header.Get("Content-Disposition"); !strings.Contains(cd, ".tar.zst") {
		t.Fatalf("expected .tar.zst filename, got %s", cd)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	entries := tarZstdEntries(t, body)
	if _, ok := entries["main.db"]; !ok {
		t.Fatalf("expected main.db in zip")
	}
	if _, ok := entries["msg.db"]; !ok {
		t.Fatalf("expected msg.db in zip")
	}
	manifestData, ok := entries["manifest.json"]
	if !ok {
		t.Fatalf("manifest missing")
	}
	manifest := parseManifest(t, manifestData)
	if len(manifest.Databases) != 2 {
		t.Fatalf("expected 2 databases, got %d", len(manifest.Databases))
	}
	if manifest.Options["db"] != "all" {
		t.Fatalf("expected manifest option db=all, got %s", manifest.Options["db"])
	}
}

func TestBackupDBScopeMain(t *testing.T) {
	server := newTestServer()
	defer server.Close()

	resp, err := http.Get(server.URL + "/backupdb?db=main")
	if err != nil {
		t.Fatalf("get backupdb: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 got %d", resp.StatusCode)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	entries := tarZstdEntries(t, body)
	if _, ok := entries["msg.db"]; ok {
		t.Fatalf("did not expect msg.db in scoped backup")
	}
	manifestData, ok := entries["manifest.json"]
	if !ok {
		t.Fatalf("manifest missing")
	}
	manifest := parseManifest(t, manifestData)
	if len(manifest.Databases) != 1 {
		t.Fatalf("expected 1 database, got %d", len(manifest.Databases))
	}
	if manifest.Databases[0].Name != "main" {
		t.Fatalf("expected main in manifest, got %s", manifest.Databases[0].Name)
	}
	if manifest.Options["db"] != "main" {
		t.Fatalf("expected manifest option db=main, got %s", manifest.Options["db"])
	}
}

func TestBackupDBInvalidSelection(t *testing.T) {
	server := newTestServer()
	defer server.Close()

	resp, err := http.Get(server.URL + "/backupdb?db=unknown")
	if err != nil {
		t.Fatalf("get backupdb: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 got %d", resp.StatusCode)
	}
}

func TestBackupDBToken(t *testing.T) {
	t.Setenv(backupTokenEnvKey, "secret-token")

	server := newTestServer()
	defer server.Close()

	resp, err := http.Get(server.URL + "/backupdb")
	if err != nil {
		t.Fatalf("get backupdb without token: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401 got %d", resp.StatusCode)
	}

	req, err := http.NewRequest(http.MethodGet, server.URL+"/backupdb", nil)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	req.Header.Set("X-Backup-Token", "secret-token")
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("get backupdb with token: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 got %d", resp.StatusCode)
	}
}
