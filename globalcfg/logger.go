package g

import (
	"log/slog"
	"os"
	"sync"
)

var gLoggerMu sync.Mutex

type LoggerWithLevel struct {
	Level  *slog.LevelVar
	Logger *slog.Logger
}

func GetLogger(name string, level slog.Level) *slog.Logger {
	gLoggerMu.Lock()
	defer gLoggerMu.Unlock()
	if loggers == nil {
		loggers = make(map[string]LoggerWithLevel)
	}
	if logger, ok := loggers[name]; ok {
		return logger.Logger
	}
	lvl := &slog.LevelVar{}
	lvl.Set(configuredLogLevel(level))
	newLogger := slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{
		Level: lvl,
	})).With(slog.String("name", name))
	loggers[name] = LoggerWithLevel{
		Level:  lvl,
		Logger: newLogger,
	}
	return newLogger
}

func configuredLogLevel(fallback slog.Level) slog.Level {
	if cfg := config.Load(); cfg != nil {
		return slog.Level(cfg.LogLevel)
	}
	return fallback
}

func GetAllLoggers() map[string]LoggerWithLevel {
	gLoggerMu.Lock()
	defer gLoggerMu.Unlock()
	res := make(map[string]LoggerWithLevel, len(loggers))
	for name, logger := range loggers {
		res[name] = logger
	}
	return res
}

func SetLoggerLevel(name string, level slog.Level) bool {
	gLoggerMu.Lock()
	defer gLoggerMu.Unlock()
	logger, ok := loggers[name]
	if !ok {
		return false
	}
	logger.Level.Set(level)
	return true
}

func SetAllLoggerLevels(level slog.Level) {
	gLoggerMu.Lock()
	defer gLoggerMu.Unlock()
	for _, logger := range loggers {
		logger.Level.Set(level)
	}
}
