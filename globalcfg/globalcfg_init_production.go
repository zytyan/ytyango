//go:build !test

package g

import (
	"context"
	"main/globalcfg/msgs"
	"main/globalcfg/q"
	"testing"
	"time"

	"go.uber.org/zap"
)

func initByConfig() {
	var err error
	config = initConfig()
	gWriteSyncer = initWriteSyncer()
	db = initDatabase(config.DatabasePath)
	logger := GetLogger("database", zap.WarnLevel)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	Q, err = q.PrepareWithLogger(ctx, db, logger.Desugar())
	if err != nil {
		panic(err)
	}
	// 设定50ms为慢查询的基线，这对嵌入式的SQLite算是慢的了，说明可能有性能抖动或查询本身有问题。
	Q.SlowQueryThreshold = time.Millisecond * 50
	err = Q.BuildCountByRatePrefixSum()
	if err != nil {
		panic(err)
	}
	logger.Infof("Database main initialized")

	msgDb = initDatabase(config.MsgDbPath)
	logger = GetLogger("msgs_database", zap.WarnLevel)
	ctx, cancel = context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	Msgs, err = msgs.PrepareWithLogger(ctx, msgDb, logger.Desugar())
	if err != nil {
		panic(err)
	}
	Msgs.SlowQueryThreshold = time.Millisecond * 50
	logger.Infof("Database msgs initialized")
}

func init() {
	if testing.Testing() {
		return
	}
	initByConfig()
}
