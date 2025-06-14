package globalcfg

import (
	"fmt"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
	"gopkg.in/yaml.v3"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"main/helpers/azure"
	"moul.io/zapgorm2"
	"os"
	"sync"
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
type MeiliConfig struct {
	BaseUrl        string `yaml:"base-url"`
	IndexName      string `yaml:"index-name"`
	PrimaryKey     string `yaml:"primary-key"`
	MasterKey      string `yaml:"master-key,omitempty"`
	saveUrlCache   string
	searchUrlCache string
}
type SeseThreshold struct {
	AdultThreshold float64 `yaml:"adult-threshold"`
	RacyThreshold  float64 `yaml:"racy-threshold"`
}

func (m *MeiliConfig) GetSaveUrl() string {
	if m.saveUrlCache != "" {
		return m.saveUrlCache
	}
	m.saveUrlCache = fmt.Sprintf("%s/indexes/%s/documents?primaryKey=%s", m.BaseUrl, m.IndexName, m.PrimaryKey)
	return m.saveUrlCache
}
func (m *MeiliConfig) GetSearchUrl() string {
	if m.searchUrlCache != "" {
		return m.searchUrlCache
	}
	m.searchUrlCache = fmt.Sprintf("%s/indexes/%s/search", m.BaseUrl, m.IndexName)
	return m.searchUrlCache
}

type Config struct {
	BotToken           string        `yaml:"bot-token"`
	God                int64         `yaml:"god"`
	MeiliConfig        MeiliConfig   `yaml:"meili-config"`
	ContentModerator   Azure         `yaml:"content-moderator"`
	Ocr                OcrConfig     `yaml:"ocr"`
	QrScanUrl          string        `yaml:"qr-scan-url"`
	SaveMessage        bool          `yaml:"save-message"`
	TgApiUrl           string        `yaml:"tg-api-url"`
	DropPendingUpdates bool          `yaml:"drop-pending-updates"`
	SeseThreshold      SeseThreshold `yaml:"sese"`
	LogLevel           int8          `yaml:"log-level"`
	LocalKvDbPath      string        `yaml:"local-kv-db-path"`
	TmpPath            string        `yaml:"tmp-path"`
	DatabasePath       string        `yaml:"database-path"`
	GeminiKey          string        `yaml:"gemini-key"`
}

var Ocr *azure.Ocr = nil
var Moderator *azure.Moderator = nil
var loggers = make(map[string]LoggerWithLevel)
var globalMu sync.Mutex

var GetConfig = sync.OnceValue[*Config](func() *Config {
	var config Config
	configFile, exists := os.LookupEnv("GOYTYAN_CONFIG")
	if !exists {
		return &config
	}
	data, err := os.ReadFile(configFile)
	if err != nil {
		panic(err)
	}
	err = yaml.Unmarshal(data, &config)
	if err != nil {
		panic(err)
	}
	Ocr = &azure.Ocr{
		Client:   *azure.NewClient(config.Ocr.Endpoint, config.Ocr.ApiKey, azure.OcrPath),
		ApiVer:   config.Ocr.ApiVer,
		Language: config.Ocr.Language,
		Features: config.Ocr.Features,
	}
	Moderator = &azure.Moderator{
		Client: *azure.NewClient(config.ContentModerator.Endpoint, config.ContentModerator.ApiKey, azure.ContentModeratorPath),
	}
	return &config
})

var gWriteSyncer = sync.OnceValue(func() zapcore.WriteSyncer {
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
})()

type LoggerWithLevel struct {
	Level  *zap.AtomicLevel
	Logger *zap.SugaredLogger
}

func GetLogger(name string) *zap.SugaredLogger {
	globalMu.Lock()
	defer globalMu.Unlock()
	if logger, ok := loggers[name]; ok {
		return logger.Logger
	}
	lvl := zap.NewAtomicLevel()
	lvl.SetLevel(zapcore.Level(GetConfig().LogLevel))
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

func GetAllLoggers() map[string]LoggerWithLevel {
	return loggers
}

var GetDb = sync.OnceValue(func() *gorm.DB {
	var err error
	newLogger := zapgorm2.New(GetLogger("gorm").Desugar())
	newLogger.IgnoreRecordNotFoundError = true
	db, err := gorm.Open(sqlite.Open(GetConfig().DatabasePath), &gorm.Config{
		Logger:                                   newLogger,
		DisableForeignKeyConstraintWhenMigrating: false,
	})
	if err != nil {
		panic(err)
	}
	return db
})
