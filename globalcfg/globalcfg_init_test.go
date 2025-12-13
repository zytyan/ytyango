package g

import (
	"context"
	"main/globalcfg/msgs"
	"main/globalcfg/q"
	"testing"
	"time"
)

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
	initMainDatabaseInMemory(db)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	Q, err = q.PrepareWithLogger(ctx, db, logger.Desugar())
	if err != nil {
		panic(err)
	}
	Msgs, err = msgs.PrepareWithLogger(ctx, db, logger.Desugar())
	if err != nil {
		panic(err)
	}
	logger.Infof("Database initialized")

}
