package q

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math/rand/v2"
	"sort"
	"sync"
	"time"

	"github.com/mattn/go-sqlite3"
)

var psMu sync.RWMutex
var rateCounterRefreshOnce sync.Once

const (
	minPicRate   = 0
	maxPicRate   = 7
	picRateCount = maxPicRate - minPicRate + 1
)

// prefix sum: ps[i] = rate=(minPicRate+i) 的累计图片数量
var countByRatePrefixSum []int64

func (q *Queries) BuildCountByRatePrefixSum() error {
	if err := q.rebuildCountByRatePrefixSum(); err != nil {
		return err
	}
	rateCounterRefreshOnce.Do(func() {
		go func() {
			ticker := time.NewTicker(6 * time.Hour)
			defer ticker.Stop()
			for range ticker.C {
				_ = q.rebuildCountByRatePrefixSum()
			}
		}()
	})
	return nil
}

func (q *Queries) rebuildCountByRatePrefixSum() error {
	psMu.Lock()
	defer psMu.Unlock()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	counts, err := q.listNsfwPicRateCounter(ctx)
	if err != nil {
		return err
	}
	buildPrefixSumFromSparse(counts)
	return nil
}

func buildPrefixSumFromSparse(counts []PicRateCounter) []int64 {
	histogram := make([]int64, picRateCount)
	for _, c := range counts {
		if c.Rate < minPicRate || c.Rate > maxPicRate || c.Count <= 0 {
			continue
		}
		histogram[c.Rate-minPicRate] = c.Count
	}
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

// getRandomRangeByWeight 按照每个 rate 的图片数量获取 [start, end) 中的 rate。
func getRandomRangeByWeight(start, end int) (int, error) {
	if start < minPicRate || end > maxPicRate+1 || start >= end ||
		len(countByRatePrefixSum) != picRateCount+1 {
		return 0, fmt.Errorf("invalid range: start %d end %d", start, end)
	}
	start, end = start-minPicRate, end-minPicRate
	rndStart, rndEnd := countByRatePrefixSum[start], countByRatePrefixSum[end]
	if rndStart >= rndEnd {
		return 0, fmt.Errorf("invalid range: start %d end %d", start, end)
	}
	rnd := rand.Int64N(rndEnd-rndStart) + rndStart
	idx := sort.Search(len(countByRatePrefixSum), func(i int) bool {
		return countByRatePrefixSum[i] > rnd
	})
	return idx - 1 + minPicRate, nil
}

func (q *Queries) GetPicByUserRate(ctx context.Context, rate int) (SavedPic, error) {
	rnd := int64(rand.Uint64())
	result, err := q.getNsfwPicByRateAndRandKey(ctx, int64(rate), rnd)
	if errors.Is(err, sql.ErrNoRows) {
		return q.getNsfwPicByRateFirst(ctx, int64(rate))
	}
	return result, err
}

func (q *Queries) GetPicByUserRateRange(ctx context.Context, start, end int) (save SavedPic, err error) {
	psMu.RLock()
	initialized := len(countByRatePrefixSum) == picRateCount+1
	psMu.RUnlock()
	if !initialized {
		if err = q.BuildCountByRatePrefixSum(); err != nil {
			return
		}
	}

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
		_, err := q.createOrUpdateNsfwPic(ctx,
			fileUid,
			fileId,
			int64(botRate),
			rnd,
		)

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
	rate, err := q.getNsfwPicRateByUserId(ctx, fileUid, userID)
	if errors.Is(err, sql.ErrNoRows) {
		err = q.createNsfwPicUserRate(ctx, fileUid, userID, newRate)
		return false, 0, err
	} else if err != nil {
		return false, 0, err
	}
	if rate == newRate {
		return true, rate, nil
	}
	err = q.updateNsfwPicUserRate(ctx, newRate, fileUid, userID)
	return true, rate, err
}
