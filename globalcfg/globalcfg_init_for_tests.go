package g

import (
	"context"
	"database/sql"
	"main/globalcfg/q"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
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

func initDatabaseInMemory(database *sql.DB) {
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
		_, err = database.Exec(string(data))
	}
}

func init() {
	if !testing.Testing() {
		return
	}
	if Q != nil {
		return
	}
	var err error
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
		DatabasePath:       ":memory:",
		GeminiKey:          "",
	}
	gWriteSyncer = initWriteSyncer()
	logger := GetLogger("database")
	db = initDatabase(config.DatabasePath)
	initDatabaseInMemory(db)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	Q, err = q.PrepareWithLogger(ctx, db, logger.Desugar())
	if err != nil {
		panic(err)
	}
	logger.Infof("Database initialized")
}
