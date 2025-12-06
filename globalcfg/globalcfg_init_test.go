package g

import (
	"context"
	"database/sql"
	_ "embed"
	"log"
	"main/globalcfg/q"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func initDatabaseInMemory(database *sql.DB) {
	dir, err := os.ReadDir("../sql")
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
		data, err := os.ReadFile(filepath.Join("../sql", name))
		if err != nil {
			panic(err)
		}
		log.Printf("Executing: %s\n", name)
		_, err = database.Exec(string(data))
	}
}
func init() {
	if !testing.Testing() {
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
