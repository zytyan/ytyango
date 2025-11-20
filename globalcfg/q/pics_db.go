package q

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math/rand"

	"github.com/mattn/go-sqlite3"
)

func (q *Queries) GetPicByUserRate(ctx context.Context, rate int) (string, error) {
	rnd := int64(rand.Uint64())
	result, err := q.getPicByRateAndRandKey(ctx, int64(rate), rnd)
	if !errors.Is(err, sql.ErrNoRows) {
		return result, err
	}
	return q.getPicByRateFirst(ctx, int64(rate))
}

func (q *Queries) AddPic(ctx context.Context, fileUid, fileId string, botRate int) error {
	for i := 0; i < 3; i++ {
		rnd := int64(rand.Uint64())
		err := q.insertPic(ctx, insertPicParams{
			FileUid:  fileUid,
			FileID:   fileId,
			BotRate:  int64(botRate),
			RandKey:  rnd,
			UserRate: int64(botRate),
		})
		var sqliteErr *sqlite3.Error
		if errors.As(err, &sqliteErr) && errors.Is(sqliteErr.ExtendedCode, sqlite3.ErrConstraintUnique) {
			// 是 RandKey 冲突！继续下一次循环尝试
			continue
		} else if err != nil {
			return err
		}
		return nil
	}
	return fmt.Errorf("failed to insert picture due to persistent RandKey collision after 3 attempts: fileUid=%s", fileUid)
}
