package q

import (
	"context"
	"main/helpers/lrusf"
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
	return query, nil
}

func InitCache(q *Queries) { initCaches(q) }

var cacheOnce sync.Once

func initCaches(q *Queries) {
	cacheOnce.Do(func() {
		userCache = lrusf.New[int64, *User](2048, id2str, nil)
		chatCache = lrusf.New[int64, *ChatCfg](2048, id2str, func(i int64, cfg *ChatCfg) {
			ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
			defer cancel()
			_ = cfg.Save(ctx, q)
		})
	})
}
