package q

import (
	"context"
	"main/helpers/lrusf"
	"strconv"
	"sync"
	"time"

	"go.uber.org/zap"
)

func PrepareWithLogger(ctx context.Context, db DBTX, logger *zap.Logger) (*Queries, error) {
	query, err := Prepare(ctx, db)
	if err != nil {
		return nil, err
	}
	query.logger = logger
	InitCache(query)
	return query, nil
}

func InitCache(q *Queries) { initCaches(q) }

var cacheOnce sync.Once

func chatStatCacheKey(key ChatStatKey) string {
	buf := make([]byte, 0, 17)
	buf = strconv.AppendInt(buf, key.Id, 16)
	buf = append(buf, ',')
	buf = strconv.AppendInt(buf, key.Day, 16)
	return string(buf)
}

func initCaches(q *Queries) {
	cacheOnce.Do(func() {
		userCache = lrusf.NewCache[int64, *User](2048, id2str, nil)
		chatCache = lrusf.NewCache[int64, *ChatCfg](2048, id2str, func(i int64, cfg *ChatCfg) {
			ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
			defer cancel()
			_ = cfg.Save(ctx, q)
		})
		chatStatCache = lrusf.NewCache[ChatStatKey, *ChatStat](64, chatStatCacheKey, func(key ChatStatKey, daily *ChatStat) {
			ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
			defer cancel()
			_ = daily.Save(ctx, q)
		})
	})
}
