package globalcfg

import (
	"fmt"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/yaml.v3"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"main/helpers/azure"
	"moul.io/zapgorm2"
	"os"
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
}

var config *Config = nil
var Ocr *azure.Ocr = nil
var Moderator *azure.Moderator = nil
var loggers = make(map[string]*zap.SugaredLogger)

func LoadConfig(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	err = yaml.Unmarshal(data, &config)
	if err != nil {
		return err
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
	if err != nil {
		return err
	}
	return err
}
func GetConfig() *Config {
	// safe: 初始化会在init内部，不会有并发问题
	return config
}

func GetLogger(name string) *zap.SugaredLogger {
	logCfg := zap.NewProductionConfig()
	logCfg.Level.SetLevel(zapcore.Level(GetConfig().LogLevel))
	logCfg.OutputPaths = []string{"stdout", "log.log"}
	build, err := logCfg.Build()
	if err != nil {
		panic(err)
	}
	logger := build.Sugar()
	loggers[name] = logger
	return logger
}

func init() {
	path, exists := os.LookupEnv("CONFIG_PATH")
	if !exists {
		path = "config.yaml"
	}
	err := LoadConfig(path)
	if err != nil {
		panic(err)
	}
	initDB()
}

var db *gorm.DB

func initDB() {
	var err error
	newLogger := zapgorm2.New(GetLogger("gorm").Desugar())
	newLogger.IgnoreRecordNotFoundError = true
	db, err = gorm.Open(sqlite.Open(config.DatabasePath), &gorm.Config{
		Logger: newLogger,

		DisableForeignKeyConstraintWhenMigrating: false,
	})
	if err != nil {
		panic(err)
	}
}

func GetDb() *gorm.DB {
	return db
}
