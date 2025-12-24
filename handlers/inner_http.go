package handlers

import (
	"archive/zip"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io"
	g "main/globalcfg"
	"net"
	"net/http"
	"net/http/pprof"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	json "github.com/json-iterator/go"
	sqlite3 "github.com/mattn/go-sqlite3"
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
	backupTokenEnvKey    = "GOYTYAN_BACKUP_TOKEN"
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

type backupSelection struct {
	includeMain bool
	includeMsg  bool
	raw         string
}

type backupManifestDB struct {
	Name string `json:"name"`
	Path string `json:"path"`
	Size int64  `json:"size"`
}

type backupManifest struct {
	Timestamp time.Time          `json:"timestamp"`
	Databases []backupManifestDB `json:"databases"`
	Options   map[string]string  `json:"options"`
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

func parseBackupSelection(param string) (backupSelection, error) {
	value := strings.ToLower(strings.TrimSpace(param))
	if value == "" {
		value = "all"
	}
	switch value {
	case "all":
		return backupSelection{includeMain: true, includeMsg: true, raw: "all"}, nil
	case "main":
		return backupSelection{includeMain: true, raw: "main"}, nil
	case "msg":
		return backupSelection{includeMsg: true, raw: "msg"}, nil
	default:
		return backupSelection{}, fmt.Errorf("invalid db query: %s", param)
	}
}

func checkBackupToken(r *http.Request, token string) bool {
	if token == "" {
		return true
	}
	if q := r.URL.Query().Get("token"); q != "" && q == token {
		return true
	}
	return r.Header.Get("X-Backup-Token") == token
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
	g.Q.ChatStatNow(marsInfo.GroupID).IncMarsCount(marsInfo.MarsCount)
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
		g.Q.ChatStatNow(dioBanUser.GroupId).IncDioAddUserCount()
	case DioBanActionBanByWrongButton, DioBanActionBanByNoButton, DioBanActionBanByNoMsg:
		g.Q.ChatStatNow(dioBanUser.GroupId).IncDioBanUserCount()
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
		_, _ = fmt.Fprintf(w, "logger %s not found\n%s", loggerName, formatLoggers())
		return
	}

	newLevel, err := strconv.ParseInt(levelParam, 10, 8)
	if err != nil {
		_, _ = w.Write([]byte(err.Error()))
	}
	logger.Level.SetLevel(zapcore.Level(newLevel))
}

func backupSQLiteDB(ctx context.Context, src *sql.DB, dstPath string) error {
	dstDB, err := sql.Open("sqlite3", dstPath)
	if err != nil {
		return err
	}
	defer dstDB.Close()

	srcConn, err := src.Conn(ctx)
	if err != nil {
		return err
	}
	defer srcConn.Close()

	dstConn, err := dstDB.Conn(ctx)
	if err != nil {
		return err
	}
	defer dstConn.Close()

	return dstConn.Raw(func(dest interface{}) error {
		destSQLite, ok := dest.(*sqlite3.SQLiteConn)
		if !ok {
			return errors.New("unexpected destination connection type")
		}
		return srcConn.Raw(func(source interface{}) error {
			srcSQLite, ok := source.(*sqlite3.SQLiteConn)
			if !ok {
				return errors.New("unexpected source connection type")
			}
			backup, err := destSQLite.Backup("main", srcSQLite, "main")
			if err != nil {
				return err
			}
			for {
				if ctx.Err() != nil {
					_ = backup.Finish()
					return ctx.Err()
				}
				done, err := backup.Step(128)
				if err != nil {
					_ = backup.Finish()
					return err
				}
				if done {
					return backup.Finish()
				}
				time.Sleep(10 * time.Millisecond)
			}
		})
	})
}

func addFileToZip(zipWriter *zip.Writer, name, srcPath string) error {
	file, err := os.Open(srcPath)
	if err != nil {
		return err
	}
	defer file.Close()

	header := &zip.FileHeader{
		Name:   name,
		Method: zip.Deflate,
	}
	header.SetModTime(time.Now())

	writer, err := zipWriter.CreateHeader(header)
	if err != nil {
		return err
	}
	_, err = io.Copy(writer, file)
	return err
}

func writeManifest(zipWriter *zip.Writer, manifest backupManifest) error {
	data, err := json.Marshal(manifest)
	if err != nil {
		return err
	}
	header := &zip.FileHeader{
		Name:   "manifest.json",
		Method: zip.Deflate,
	}
	header.SetModTime(time.Now())
	writer, err := zipWriter.CreateHeader(header)
	if err != nil {
		return err
	}
	_, err = writer.Write(data)
	return err
}

func backupDBHandler(logger *zap.Logger) http.HandlerFunc {
	type backupTarget struct {
		name     string
		path     string
		db       *sql.DB
		destPath string
	}
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.NotFound(w, r)
			return
		}

		token := os.Getenv(backupTokenEnvKey)
		if !checkBackupToken(r, token) {
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte("unauthorized"))
			return
		}

		selection, err := parseBackupSelection(r.URL.Query().Get("db"))
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		cfg := g.GetConfig()
		if cfg == nil {
			http.Error(w, "config not initialized", http.StatusInternalServerError)
			return
		}

		targets := make([]backupTarget, 0, 2)
		if selection.includeMain {
			targets = append(targets, backupTarget{
				name: "main",
				path: cfg.DatabasePath,
				db:   g.RawMainDb(),
			})
		}
		if selection.includeMsg {
			targets = append(targets, backupTarget{
				name: "msg",
				path: cfg.MsgDbPath,
				db:   g.RawMsgsDb(),
			})
		}
		for _, target := range targets {
			if target.db == nil {
				http.Error(w, "database not initialized", http.StatusInternalServerError)
				return
			}
		}

		tmpDir, err := os.MkdirTemp("", "backupdb-*")
		if err != nil {
			logger.Error("create backup temp dir", zap.Error(err))
			http.Error(w, "failed to create temp dir", http.StatusInternalServerError)
			return
		}
		defer os.RemoveAll(tmpDir)

		ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
		defer cancel()

		start := time.Now()
		manifest := backupManifest{
			Timestamp: start.UTC(),
			Options:   map[string]string{"db": selection.raw},
		}

		for i := range targets {
			targets[i].destPath = filepath.Join(tmpDir, targets[i].name+".db")
			if err := backupSQLiteDB(ctx, targets[i].db, targets[i].destPath); err != nil {
				logger.Error("backup sqlite database", zap.String("db", targets[i].name), zap.Error(err))
				http.Error(w, "backup failed", http.StatusInternalServerError)
				return
			}
			info, err := os.Stat(targets[i].destPath)
			if err != nil {
				logger.Error("stat backup file", zap.String("file", targets[i].destPath), zap.Error(err))
				http.Error(w, "backup failed", http.StatusInternalServerError)
				return
			}
			manifest.Databases = append(manifest.Databases, backupManifestDB{
				Name: targets[i].name,
				Path: targets[i].path,
				Size: info.Size(),
			})
		}

		filename := fmt.Sprintf("backup-%s.zip", manifest.Timestamp.Format("20060102-150405Z"))
		w.Header().Set("Content-Type", "application/zip")
		w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))

		zipWriter := zip.NewWriter(w)
		for _, target := range targets {
			if err := addFileToZip(zipWriter, target.name+".db", target.destPath); err != nil {
				logger.Error("write backup zip entry", zap.String("db", target.name), zap.Error(err))
				return
			}
		}
		if err := writeManifest(zipWriter, manifest); err != nil {
			logger.Error("write backup manifest", zap.Error(err))
			return
		}
		if err := zipWriter.Close(); err != nil {
			logger.Error("close backup zip", zap.Error(err))
		}

		logger.Info("backup completed",
			zap.String("filename", filename),
			zap.Duration("duration", time.Since(start)),
			zap.Any("databases", manifest.Databases),
		)
	}
}

func listAllRoutes(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	_, _ = w.Write([]byte("GET /loggers\nPUT /loggers/<name>/<:level,int8>\nGET /backupdb\n"))
}

func withLoggingAndRecovery(logger *zap.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		recorder := &statusRecorder{ResponseWriter: w}
		start := time.Now()

		defer func() {
			if rec := recover(); rec != nil {
				if recorder.status == 0 {
					recorder.WriteHeader(http.StatusInternalServerError)
				}
				logger.Error("inner http panic",
					zap.Any("panic", rec),
					zap.Stack("stack"),
				)
			}

			if recorder.status == 0 {
				recorder.status = http.StatusOK
			}

			logger.Info("inner http request",
				zap.String("method", r.Method),
				zap.String("path", r.URL.Path),
				zap.Int("status", recorder.status),
				zap.Duration("duration", time.Since(start)),
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

func buildHandler(logger *zap.Logger) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/mars-counter", marsCounter)
	mux.HandleFunc("/dio-ban", dioBan)
	mux.HandleFunc("/backupdb", backupDBHandler(logger))
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
	logger := g.GetLogger("inner-http", zap.WarnLevel)
	addr, enabled := resolveInnerHTTPAddr(os.Getenv(innerHTTPEnvKey))
	if !enabled {
		logger.Infof("%s=OFF, inner http server disabled", innerHTTPEnvKey)
		return
	}

	handler := buildHandler(logger.Desugar())
	server := &http.Server{
		Addr:              addr,
		Handler:           handler,
		ReadHeaderTimeout: 3 * time.Second,
	}

	logger.Infof("inner http server listening on %s", addr)
	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		logger.Fatalf("inner http server error: %s", err)
	}
}
