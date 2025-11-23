package q

import (
	"cmp"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math/rand/v2"
	"slices"
	"sort"
	"sync"
	"time"

	"github.com/mattn/go-sqlite3"
	"go.uber.org/zap"
)

var psMu sync.RWMutex

// prefix sum: ps[i] = rate=(minRate+i) 的累计数量
var countByRatePrefixSum []int64
var minCountRate int // prefixSum 的偏移量

func (q *Queries) BuildCountByRatePrefixSum() error {
	psMu.Lock()
	defer psMu.Unlock()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	counts, err := q.getPicRateCounts(ctx)
	if err != nil {
		return err
	}
	buildPrefixSumFromSparse(counts)
	go func() {
		for range time.Tick(6 * time.Hour) {
			err := q.BuildCountByRatePrefixSum()
			if err != nil {
				q.logger.Warn("rebuild BuildCountByRatePrefixSum failed", zap.Error(err))
			}
		}
	}()
	return nil
}

func buildPrefixSumFromSparse(counts []PicRateCounter) []int64 {
	slices.SortFunc(counts, func(a, b PicRateCounter) int {
		return cmp.Compare(a.Rate, b.Rate)
	})

	minRate := counts[0].Rate
	maxRate := counts[len(counts)-1].Rate
	size := maxRate - minRate + 1
	histogram := make([]int64, size)
	for _, c := range counts {
		histogram[c.Rate-minRate] = c.Count
	}
	minCountRate = int(minRate)
	countByRatePrefixSum = buildPrefixSum(histogram)
	return countByRatePrefixSum
}

func buildPrefixSum(counts []int64) []int64 {
	r := make([]int64, len(counts)+1)
	r[0] = 0
	for i := range counts {
		r[i+1] = r[i] + counts[i]
	}
	return r
}

// getRandomRangeByWeight 按照rate中的数量获取 [start, end)
func getRandomRangeByWeight(start, end int) (int, error) {
	if start < minCountRate || end < minCountRate || start > end ||
		start > len(countByRatePrefixSum) ||
		end > len(countByRatePrefixSum) {
		return 0, fmt.Errorf("invalid range: start %d end %d", start, end)
	}
	start, end = start-minCountRate, end-minCountRate
	rndStart, rndEnd := countByRatePrefixSum[start], countByRatePrefixSum[end]
	rnd := rand.Int64N(rndEnd-rndStart) + rndStart
	idx := sort.Search(len(countByRatePrefixSum), func(i int) bool {
		return countByRatePrefixSum[i] > rnd
	})
	return idx - 1 + minCountRate, nil
}

func (q *Queries) GetPicByUserRate(ctx context.Context, rate int) (SavedPic, error) {
	rnd := int64(rand.Uint64())
	result, err := q.getPicByRateAndRandKey(ctx, int64(rate), rnd)
	if errors.Is(err, sql.ErrNoRows) {
		return q.getPicByRateFirst(ctx, int64(rate))
	}
	return result, err
}

func (q *Queries) GetPicByUserRateRange(ctx context.Context, start, end int) (save SavedPic, err error) {
	psMu.RLock()
	defer psMu.RUnlock()
	rate, err := getRandomRangeByWeight(start, end)
	if err != nil {
		return
	}
	return q.GetPicByUserRate(ctx, rate)
}

func (q *Queries) AddPic(ctx context.Context, fileUid, fileId string, botRate int) error {
	psMu.Lock()
	defer psMu.Unlock()
	for i := 0; i < 3; i++ {
		rnd := int64(rand.Uint64())
		_, err := q.insertPic(ctx, insertPicParams{
			FileUid:  fileUid,
			FileID:   fileId,
			BotRate:  int64(botRate),
			RandKey:  rnd,
			UserRate: int64(botRate),
		})

		// RandKey 冲突 → 重试
		var sErr *sqlite3.Error
		if errors.As(err, &sErr) &&
			errors.Is(sErr.Code, sqlite3.ErrConstraint) &&
			errors.Is(sErr.ExtendedCode, sqlite3.ErrConstraintUnique) {
			continue
		}
		if err != nil {
			return err
		}
		return nil
	}
	return fmt.Errorf("failed to insert picture due to persistent RandKey collision after 3 attempts: fileUid=%s", fileUid)
}

func (q *Queries) RatePic(ctx context.Context, fileUid string, userID int64, newRate int64) (bool, int64, error) {
	rate, err := q.getPicRateByUserId(ctx, fileUid, userID)
	if errors.Is(err, sql.ErrNoRows) {
		err = q.ratePic(ctx, fileUid, userID, newRate)
		return false, 0, err
	} else if err != nil {
		return false, 0, err
	}
	if rate == newRate {
		return true, rate, nil
	}
	err = q.updatePicRate(ctx, newRate, fileUid, userID)
	return true, rate, err
}
