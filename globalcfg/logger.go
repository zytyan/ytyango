package g

import (
	"log/slog"
	"os"
	"sync"
)

var gLoggerMu sync.Mutex
var gDefaultLogger *slog.Logger

type LoggerWithLevel struct {
	Level  *slog.LevelVar
	Logger *slog.Logger
}

func GetLogger(name string, level slog.Level) *slog.Logger {
	gLoggerMu.Lock()
	defer gLoggerMu.Unlock()
	if gDefaultLogger == nil {
		gDefaultLogger = slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{}))
	}
	if loggers == nil {
		loggers = make(map[string]LoggerWithLevel)
	}
	if logger, ok := loggers[name]; ok {
		return logger.Logger
	}
	lvl := &slog.LevelVar{}
	lvl.Set(level)
	newLogger := gDefaultLogger.With(slog.String("name", name))
	loggers[name] = LoggerWithLevel{
		Level:  lvl,
		Logger: newLogger,
	}
	return newLogger
}
