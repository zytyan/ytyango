package g

import (
	"os"
	"sync"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

var gLoggerMu sync.Mutex
var gWriteSyncer zapcore.WriteSyncer

func initWriteSyncer() zapcore.WriteSyncer {
	cfg := GetConfig()
	logfile := cfg.LogFile
	if logfile == "" {
		return zapcore.AddSync(os.Stderr)
	}
	w := zapcore.AddSync(&lumberjack.Logger{
		Filename:   logfile,
		MaxSize:    1, // megabytes
		MaxBackups: 10,
		MaxAge:     100, // days
		Compress:   true,
		LocalTime:  true,
	})
	if !cfg.NoStdout {
		w = zapcore.NewMultiWriteSyncer(w, zapcore.AddSync(os.Stdout))
	}
	return w
}

type LoggerWithLevel struct {
	Level  *zap.AtomicLevel
	Logger *zap.SugaredLogger
}

func GetLogger(name string, level zapcore.Level) *zap.SugaredLogger {
	gLoggerMu.Lock()
	defer gLoggerMu.Unlock()
	if loggers == nil {
		loggers = make(map[string]LoggerWithLevel)
	}
	if logger, ok := loggers[name]; ok {
		return logger.Logger
	}
	lvl := zap.NewAtomicLevel()
	lvl.SetLevel(level)
	core := zapcore.NewCore(
		zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig()),
		gWriteSyncer,
		lvl,
	)
	logger := zap.New(core)
	newLogger := logger.
		With(zap.String("name", name)).
		Sugar()
	loggers[name] = LoggerWithLevel{
		Level:  &lvl,
		Logger: newLogger,
	}
	return newLogger
}
