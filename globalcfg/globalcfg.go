package g

import (
	"database/sql"
	"main/globalcfg/q"
	"main/helpers/azure"
	"os"
	"sync"

	_ "github.com/mattn/go-sqlite3"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
	"gopkg.in/yaml.v3"
)

type Azure struct {
	Endpoint string `yaml:"endpoint"`
	ApiKey   string `yaml:"api-key"`
}
type OcrConfig struct {
	Azure    `yaml:",inline"`
	ApiVer   string `yaml:"api-ver"`
	Language string `yaml:"language"`
	Features string `yaml:"features"`
}

type Config struct {
	BotToken           string      `yaml:"bot-token"`
	God                int64       `yaml:"god"`
	MeiliConfig        MeiliConfig `yaml:"meili-config"`
	ContentModerator   Azure       `yaml:"content-moderator"`
	Ocr                OcrConfig   `yaml:"ocr"`
	QrScanUrl          string      `yaml:"qr-scan-url"`
	SaveMessage        bool        `yaml:"save-message"`
	TgApiUrl           string      `yaml:"tg-api-url"`
	DropPendingUpdates bool        `yaml:"drop-pending-updates"`
	LogLevel           int8        `yaml:"log-level"`
	LocalKvDbPath      string      `yaml:"local-kv-db-path"`
	TmpPath            string      `yaml:"tmp-path"`
	DatabasePath       string      `yaml:"database-path"`
	GeminiKey          string      `yaml:"gemini-key"`
}

var Ocr *azure.Ocr
var Moderator *azure.ModeratorV2
var loggers = make(map[string]LoggerWithLevel)
var gLoggerMu sync.Mutex
var config *Config

func initConfig() *Config {
	var cfg Config
	configFile, exists := os.LookupEnv("GOYTYAN_CONFIG")
	if !exists {
		return &cfg
	}
	data, err := os.ReadFile(configFile)
	if err != nil {
		panic(err)
	}
	err = yaml.Unmarshal(data, &cfg)
	if err != nil {
		panic(err)
	}
	Ocr = &azure.Ocr{
		Client:   *azure.NewClient(cfg.Ocr.Endpoint, cfg.Ocr.ApiKey, azure.OcrPath),
		ApiVer:   cfg.Ocr.ApiVer,
		Language: cfg.Ocr.Language,
		Features: cfg.Ocr.Features,
	}
	Moderator = &azure.ModeratorV2{
		Client:     *azure.NewClient(cfg.ContentModerator.Endpoint, cfg.ContentModerator.ApiKey, azure.ContentModeratorV2Path),
		Categories: []string{azure.ModerateV2CatSexual}, // 也只查涩图
		OutputType: "FourSeverityLevels",                // Azure仅支持这个参数，所以硬编码
	}
	return &cfg
}

var gWriteSyncer zapcore.WriteSyncer

func initWriteSyncer() zapcore.WriteSyncer {
	logfile, exists := os.LookupEnv("GOYTYAN_LOG_FILE")
	if !exists {
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
	_, noStdout := os.LookupEnv("GOYTYAN_NO_STDOUT")
	if !noStdout {
		w = zapcore.NewMultiWriteSyncer(w, zapcore.AddSync(os.Stdout))
	}
	return w
}

type LoggerWithLevel struct {
	Level  *zap.AtomicLevel
	Logger *zap.SugaredLogger
}

func GetLogger(name string) *zap.SugaredLogger {
	gLoggerMu.Lock()
	defer gLoggerMu.Unlock()
	if logger, ok := loggers[name]; ok {
		return logger.Logger
	}
	lvl := zap.NewAtomicLevel()
	lvl.SetLevel(zapcore.Level(config.LogLevel))
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

func GetConfig() *Config {
	return config
}

func GetAllLoggers() map[string]LoggerWithLevel {
	return loggers
}

var db *sql.DB
var Q *q.Queries

func initDatabase() *sql.DB {
	dbPath := config.DatabasePath
	if dbPath == "" {
		dbPath = "ytyan_new.db"
	}
	d, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		panic(err)
	}
	d.Exec("PRAGMA journal_mode=WAL;")
	d.Exec("PRAGMA synchronous=OFF;")
	d.Exec("PRAGMA cache_size=-80000;")
	return d
}

func NewTx() (*q.Queries, error) {
	tx, err := db.Begin()
	if err != nil {
		return nil, err
	}
	return Q.WithTx(tx), nil
}

func init() {
	config = initConfig()
	gWriteSyncer = initWriteSyncer()
	db = initDatabase()
	Q = q.New(db)
}
