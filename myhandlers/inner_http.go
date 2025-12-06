package myhandlers

import (
	"encoding/json"
	"fmt"
	g "main/globalcfg"
	"net"
	"net/http"
	"net/http/pprof"
	"os"
	"runtime/debug"
	"strconv"
	"strings"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

const (
	DioBanActionAdd = iota
	DioBanActionBanByWrongButton
	DioBanActionBanByNoButton
	DioBanActionBanByNoMsg
)

const (
	innerHTTPEnvKey      = "BOT_INNER_HTTP"
	innerHTTPDefaultAddr = "127.0.0.1:4019"
	maxRequestBodyBytes  = 1 << 20 // 1MB upper bound for small JSON payloads
)

type MarsInfo struct {
	GroupID   int64 `json:"group_id"`
	MarsCount int64 `json:"mars_count"`
}

type DioBanUser struct {
	UserId  int64 `json:"user_id"`
	GroupId int64 `json:"group_id"`
	Action  int   `json:"action"`
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(status int) {
	if r.status != 0 {
		return
	}
	r.status = status
	r.ResponseWriter.WriteHeader(status)
}

func (r *statusRecorder) Write(b []byte) (int, error) {
	if r.status == 0 {
		r.status = http.StatusOK
		r.ResponseWriter.WriteHeader(http.StatusOK)
	}
	return r.ResponseWriter.Write(b)
}

func decodeJSONBody(w http.ResponseWriter, req *http.Request, v any) error {
	defer req.Body.Close()
	reader := http.MaxBytesReader(w, req.Body, maxRequestBodyBytes)
	decoder := json.NewDecoder(reader)
	return decoder.Decode(v)
}

func marsCounter(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.NotFound(w, r)
		return
	}
	var marsInfo MarsInfo
	if err := decodeJSONBody(w, r, &marsInfo); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	g.Q.ChatStatToday(marsInfo.GroupID).IncMarsCount(marsInfo.MarsCount)
	w.WriteHeader(http.StatusOK)
}

func dioBan(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.NotFound(w, r)
		return
	}
	var dioBanUser DioBanUser
	if err := decodeJSONBody(w, r, &dioBanUser); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	switch dioBanUser.Action {
	case DioBanActionAdd:
		g.Q.ChatStatToday(dioBanUser.GroupId).IncDioAddUserCount()
	case DioBanActionBanByWrongButton, DioBanActionBanByNoButton, DioBanActionBanByNoMsg:
		g.Q.ChatStatToday(dioBanUser.GroupId).IncDioBanUserCount()
	}
	w.WriteHeader(http.StatusOK)
}

func formatLoggers() string {
	buf := strings.Builder{}
	for name, logger := range g.GetAllLoggers() {
		level := logger.Level.Level()
		buf.WriteString(
			fmt.Sprintf("%-16s\t[%d]%s\n", name, level, level.String()),
		)
	}
	return buf.String()
}

func showLoggers(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(formatLoggers()))
}

func parseLoggerParams(path string) (string, string, bool) {
	trimmed := strings.TrimPrefix(path, "/loggers/")
	parts := strings.Split(trimmed, "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", false
	}
	return parts[0], parts[1], true
}

func setLoggerLevel(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		http.NotFound(w, r)
		return
	}
	loggerName, levelParam, ok := parseLoggerParams(r.URL.Path)
	if !ok {
		http.NotFound(w, r)
		return
	}

	logger, exists := g.GetAllLoggers()[loggerName]
	if !exists {
		_, _ = w.Write([]byte(fmt.Sprintf("logger %s not found\n%s", loggerName, formatLoggers())))
		return
	}

	newLevel, err := strconv.ParseInt(levelParam, 10, 8)
	if err != nil {
		_, _ = w.Write([]byte(err.Error()))
	}
	logger.Level.SetLevel(zapcore.Level(newLevel))
}

func listAllRoutes(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	_, _ = w.Write([]byte("GET /loggers\nPUT /loggers/<name>/<:level,int8>\n"))
}

func withLoggingAndRecovery(logger *zap.SugaredLogger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		recorder := &statusRecorder{ResponseWriter: w}
		start := time.Now()
		defer func() {
			if rec := recover(); rec != nil {
				if recorder.status == 0 {
					recorder.WriteHeader(http.StatusInternalServerError)
				}
				logger.Errorw("inner http panic", "panic", rec, "stack", string(debug.Stack()))
			}
			if recorder.status == 0 {
				recorder.status = http.StatusOK
			}
			logger.Infow("inner http request",
				"method", r.Method,
				"path", r.URL.Path,
				"status", recorder.status,
				"duration", time.Since(start),
			)
		}()
		next.ServeHTTP(recorder, r)
	})
}

func pprofHandlers(mux *http.ServeMux) {
	mux.HandleFunc("/debug/pprof/", pprof.Index)
	mux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
	mux.HandleFunc("/debug/pprof/profile", pprof.Profile)
	mux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
	mux.HandleFunc("/debug/pprof/trace", pprof.Trace)
}

func buildHandler(logger *zap.SugaredLogger) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/mars-counter", marsCounter)
	mux.HandleFunc("/dio-ban", dioBan)
	mux.HandleFunc("/loggers", showLoggers)
	mux.HandleFunc("/loggers/", setLoggerLevel)
	mux.HandleFunc("/", listAllRoutes)
	pprofHandlers(mux)
	return withLoggingAndRecovery(logger, mux)
}

func resolveInnerHTTPAddr(envValue string) (addr string, enabled bool) {
	if strings.EqualFold(envValue, "OFF") {
		return "", false
	}
	if strings.TrimSpace(envValue) == "" {
		return innerHTTPDefaultAddr, true
	}
	if _, err := net.ResolveTCPAddr("tcp", envValue); err != nil {
		panic(err)
	}
	return envValue, true
}

func HttpListen4019() {
	logger := g.GetLogger("yt-dlp")
	addr, enabled := resolveInnerHTTPAddr(os.Getenv(innerHTTPEnvKey))
	if !enabled {
		logger.Infof("%s=OFF, inner http server disabled", innerHTTPEnvKey)
		return
	}

	handler := buildHandler(logger)
	server := &http.Server{
		Addr:              addr,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
	}

	logger.Infof("inner http server listening on %s", addr)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		logger.Fatalf("inner http server error: %s", err)
	}
}
