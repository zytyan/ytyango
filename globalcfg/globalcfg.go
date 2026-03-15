package g

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"main/globalcfg/msgs"
	"main/globalcfg/q"
	"main/helpers/azure"
	"main/helpers/meilisearch"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/v2"
	_ "github.com/mattn/go-sqlite3"
	"go.uber.org/zap/zapcore"
)

type Azure struct {
	Endpoint string `koanf:"endpoint"`
	ApiKey   string `koanf:"api-key"`
}

type OcrConfig struct {
	Azure    `koanf:",squash"`
	ApiVer   string `koanf:"api-ver"`
	Language string `koanf:"language"`
	Features string `koanf:"features"`
}

type MeiliConfig struct {
	BaseUrl    string `koanf:"base-url"`
	IndexName  string `koanf:"index-name"`
	PrimaryKey string `koanf:"primary-key"`
	MasterKey  string `koanf:"master-key"`
}

type Config struct {
	BotToken           string      `koanf:"bot-token"`
	God                int64       `koanf:"god"`
	MyChats            []int64     `koanf:"my-chats"`
	AIChats            []int64     `koanf:"ai-chats"`
	MeiliConfig        MeiliConfig `koanf:"meili-config"`
	ContentModerator   Azure       `koanf:"content-moderator"`
	Ocr                OcrConfig   `koanf:"ocr"`
	QrScanUrl          string      `koanf:"qr-scan-url"`
	SaveMessage        bool        `koanf:"save-message"`
	TgApiUrl           string      `koanf:"tg-api-url"`
	DropPendingUpdates bool        `koanf:"drop-pending-updates"`
	LogLevel           int8        `koanf:"log-level"`
	DatabasePath       string      `koanf:"database-path"`
	GeminiKey          string      `koanf:"gemini-key"`
	MsgDbPath          string      `koanf:"msg-db-path"`

	LogFile  string `koanf:"log-file"`
	NoStdout bool   `koanf:"no-stdout"`
}

var gMu sync.Mutex
var config atomic.Pointer[Config]

type PtrLinkedCfg[T any] struct {
	cfg     *Config
	ptr     *T
	fn      func(new *Config) *T
	checker func(old, new *Config) bool
}

func (p *PtrLinkedCfg[T]) Get() *T {
	cfg := GetConfig()
	if p.ptr == nil || p.cfg != cfg {
		gMu.Lock()
		defer gMu.Unlock()
		if p.checker(p.cfg, cfg) {
			p.cfg = cfg
			p.ptr = p.fn(p.cfg)
		}
	}
	return p.ptr
}
func NewPtrLinkedCfg[T any](checker func(old, new *Config) bool, getter func(new *Config) *T) PtrLinkedCfg[T] {
	return PtrLinkedCfg[T]{
		fn: getter,
		checker: func(old, new *Config) bool {
			if new == nil {
				return false
			}
			if old == nil {
				return true
			}
			return checker(old, new)
		},
	}
}

var ocr = NewPtrLinkedCfg(
	func(old, new *Config) bool {
		return old.Ocr != new.Ocr
	},
	func(new *Config) *azure.Ocr {
		return &azure.Ocr{
			Client: *azure.NewClient(
				new.Ocr.Endpoint,
				new.Ocr.ApiKey,
				azure.OcrPath,
			),
			ApiVer:   new.Ocr.ApiVer,
			Language: new.Ocr.Language,
			Features: new.Ocr.Features,
		}
	},
)

var moderator = NewPtrLinkedCfg(
	func(old, new *Config) bool {
		return old.ContentModerator != new.ContentModerator
	},
	func(new *Config) *azure.ModeratorV2 {
		return &azure.ModeratorV2{
			Client: *azure.NewClient(
				new.ContentModerator.Endpoint,
				new.ContentModerator.ApiKey,
				azure.ContentModeratorV2Path,
			),
		}
	},
)

var meili = NewPtrLinkedCfg(
	func(old, new *Config) bool {
		return old.MeiliConfig != new.MeiliConfig
	},
	func(new *Config) *meilisearch.Client {
		return meilisearch.NewMeiliClient(
			new.MeiliConfig.BaseUrl,
			new.MeiliConfig.IndexName,
			new.MeiliConfig.MasterKey,
		)
	},
)

var loggers = make(map[string]LoggerWithLevel)

func loadConfig(k *koanf.Koanf, provider *file.File) (*Config, error) {
	err := k.Load(provider, yaml.Parser())
	if err != nil {
		return nil, err
	}
	var newCfg Config
	err = k.Unmarshal("", &newCfg)
	if err != nil {
		return nil, err
	}
	return &newCfg, nil
}

func getCfgFilename() string {
	cfgFile := os.Getenv("YTYAN_CONFIG_FILE")
	if cfgFile == "" {
		if testing.Testing() {
			return filepath.Join(mustGetProjectRootDir(), "config.example.yaml")
		}
		cfgFile = "config.yaml"
	}
	return cfgFile
}

func InitConfig() {
	k := koanf.New(".")
	cfgFile := getCfgFilename()
	provider := file.Provider(cfgFile)
	cfg, err := loadConfig(k, provider)
	if err != nil {
		panic(fmt.Sprintf("load config file failed: %v", err))
	}
	err = provider.Watch(func(event any, err error) {
		if err != nil {
			log.Printf("watch error: %v", err)
			return
		}
		cfg2, err := loadConfig(k, provider)
		if err != nil {
			log.Printf("load config file failed: %v", err)
			return
		}
		oldCfg := config.Swap(cfg2)
		if oldCfg.DatabasePath != cfg2.DatabasePath || oldCfg.MsgDbPath != cfg2.MsgDbPath {
			log.Printf("database path cannot be changed without restart, old: %s, new: %s", oldCfg.DatabasePath, cfg2.DatabasePath)
			log.Printf("message database path cannot be changed without restart, old: %s, new: %s", oldCfg.MsgDbPath, cfg2.MsgDbPath)
			return
		}
		log.Printf("config changed at %s", time.Now())
	})
	if err != nil {
		panic(err)
	}
	config.Store(cfg)
	gWriteSyncer = initWriteSyncer()
	db = getSqliteConn(config.Load().DatabasePath)
	msgDb = getSqliteConn(config.Load().MsgDbPath)
	if testing.Testing() {
		initMainDatabaseInMemory(db)
		_ = msgDb.Close()
		msgDb = db
	}
	Q, err = q.PrepareWithLogger(context.Background(), db, GetLogger("sql", zapcore.DebugLevel).Desugar())
	if err != nil {
		panic(err)
	}
	Msgs, err = msgs.PrepareWithLogger(context.Background(), msgDb, GetLogger("sql", zapcore.DebugLevel).Desugar())
	if err != nil {
		panic(err)
	}
}

func GetConfig() *Config {
	return config.Load()
}

func Ocr() *azure.Ocr {
	return ocr.Get()
}

func Moderator() *azure.ModeratorV2 {
	return moderator.Get()
}

func Meili() *meilisearch.Client {
	return meili.Get()
}

func GetAllLoggers() map[string]LoggerWithLevel {
	return loggers
}

var db *sql.DB
var msgDb *sql.DB

var Q *q.Queries
var Msgs *msgs.Queries

func getSqliteConn(dbPath string) *sql.DB {
	check := func(_ sql.Result, e error) {
		if e != nil {
			panic(e)
		}
	}
	d, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		panic(err)
	}
	check(d.Exec(`PRAGMA journal_mode=WAL;
						PRAGMA wal_autocheckpoint=1000;
						PRAGMA synchronous=NORMAL;
						PRAGMA mmap_size=67108864; -- 64MB
						PRAGMA cache_size = -32768; -- 32MB page cache
						PRAGMA busy_timeout=5000;
						PRAGMA optimize;`))
	return d
}

func RawMainDb() *sql.DB {
	return db
}

func RawMsgsDb() *sql.DB {
	return msgDb
}

func init() {
	InitConfig()
}
