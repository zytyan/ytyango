package myhandlers

import (
	"io"
	g "main/globalcfg"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"
)

func newTestServer() *httptest.Server {
	logger := g.GetLogger("inner-http-test")
	return httptest.NewServer(buildHandler(logger))
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

	stat := g.Q.ChatStatToday(groupID)
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

	stat := g.Q.ChatStatToday(groupID)
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
