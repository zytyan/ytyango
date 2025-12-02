//go:build !test

package g

import (
	"context"
	"main/globalcfg/q"
	"testing"
	"time"

	"go.uber.org/zap/zapcore"
)

func init() {
	if testing.Testing() {
		return
	}
	var err error
	config = initConfig()
	gWriteSyncer = initWriteSyncer()
	db = initDatabase(config.DatabasePath)
	logger := GetLogger("database")
	loggers["database"].Level.SetLevel(zapcore.WarnLevel)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	Q, err = q.PrepareWithLogger(ctx, db, logger.Desugar())
	if err != nil {
		panic(err)
	}
	err = Q.BuildCountByRatePrefixSum()
	if err != nil {
		panic(err)
	}
	logger.Infof("Database initialized")
}
