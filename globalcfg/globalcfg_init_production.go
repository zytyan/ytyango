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
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if config.DatabaseURL == "" {
		panic("database-url is required")
	}
	db = initPool(ctx, config.DatabaseURL)
	if err := db.Ping(ctx); err != nil {
		panic(err)
	}
	logger := GetLogger("database", zap.WarnLevel)
	Q, err = q.PrepareWithLogger(ctx, db, logger.Desugar())
	if err != nil {
		panic(err)
	}
	err = Q.BuildCountByRatePrefixSum(logger.Desugar())
	if err != nil {
		panic(err)
	}
	logger.Infof("Database main initialized")

	if config.MsgDatabaseURL == "" {
		msgDb = db
	} else {
		msgDb = initPool(ctx, config.MsgDatabaseURL)
		if err := msgDb.Ping(ctx); err != nil {
			panic(err)
		}
	}
	logger = GetLogger("msgs_database", zap.WarnLevel)
	Msgs, err = msgs.PrepareWithLogger(ctx, msgDb, logger.Desugar())
	if err != nil {
		panic(err)
	}
	logger.Infof("Database msgs initialized")
	initMeili()
}

func init() {
	if testing.Testing() {
		return
	}
	initByConfig()
}
