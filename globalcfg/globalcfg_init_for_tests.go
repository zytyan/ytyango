package g

import (
	"context"
	"main/globalcfg/msgs"
	"main/globalcfg/q"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

func mustGetProjectRootDir() string {
	current, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	for {
		parent := filepath.Dir(current)
		modFile := filepath.Join(parent, "go.mod")
		if stat, err := os.Stat(modFile); err == nil && !stat.IsDir() {
			return parent
		}
		if current == "/" {
			panic(modFile)
		}
		current = parent
	}
}

func initMainDatabase(ctx context.Context, pool *pgxpool.Pool) {
	projRoot := mustGetProjectRootDir()
	sqlDir := filepath.Join(projRoot, "sql")
	dir, err := os.ReadDir(sqlDir)
	if err != nil {
		panic(err)
	}
	for _, file := range dir {
		if file.IsDir() {
			continue
		}
		name := file.Name()
		if !strings.HasSuffix(name, ".sql") || !strings.HasPrefix(name, "schema_") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(sqlDir, name))
		if err != nil {
			panic(err)
		}
		_, err = pool.Exec(ctx, string(data))
		if err != nil {
			panic(err)
		}
	}
}

// initForTests prepares an in-memory database and schema for any go test run without loading production config.
func initForTests() {
	if !testing.Testing() {
		return
	}
	if Q != nil {
		return
	}
	var err error
	dbURL := os.Getenv("PG_TEST_URL")
	if dbURL == "" {
		dbURL = os.Getenv("DATABASE_URL")
	}
	if dbURL == "" {
		panic("PG_TEST_URL or DATABASE_URL must be set for tests")
	}
	config = &Config{
		// 此处的Token已经废弃，可放心使用
		BotToken:           "554277510:AAEKxRdcRfhEjtSIfxpaYtL19XFgdDcY23U",
		God:                0,
		MeiliConfig:        MeiliConfig{},
		ContentModerator:   Azure{},
		Ocr:                OcrConfig{},
		QrScanUrl:          "",
		SaveMessage:        false,
		TgApiUrl:           "",
		DropPendingUpdates: false,
		LogLevel:           -1, // 测试过程中打印所有日志
		LocalKvDbPath:      "",
		TmpPath:            "",
		DatabaseURL:        dbURL,
		GeminiKey:          "",
		MsgDatabaseURL:     "",
	}
	gWriteSyncer = initWriteSyncer()
	logger := GetLogger("database", zap.DebugLevel)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	db = initPool(ctx, config.DatabaseURL)
	if err := db.Ping(ctx); err != nil {
		panic(err)
	}
	msgDb = db
	initMainDatabase(ctx, db)

	ctx2, cancel2 := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel2()
	Q, err = q.PrepareWithLogger(ctx2, db, logger.Desugar())
	if err != nil {
		panic(err)
	}
	Msgs, err = msgs.PrepareWithLogger(ctx2, msgDb, logger.Desugar())
	if err != nil {
		panic(err)
	}
	logger.Infof("Database initialized for tests")
}

func init() {
	initForTests()
}
