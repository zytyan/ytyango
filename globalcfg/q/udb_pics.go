package q

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"math/rand/v2"
	"slices"
	"sort"
	"sync"
	"time"

	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"go.uber.org/zap"
)

var psMu sync.RWMutex

// prefix sum: ps[i] = rate=(minRate+i) 的累计数量
var countByRatePrefixSum []int64
var minCountRate int // prefixSum 的偏移量

func (q *Queries) BuildCountByRatePrefixSum(logger *zap.Logger) error {
	psMu.Lock()
	defer psMu.Unlock()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	counts, err := q.listNsfwPicRateCounter(ctx)
	if err != nil {
		return err
	}
	buildPrefixSumFromSparse(counts)
	go func() {
		for range time.Tick(6 * time.Hour) {
			err := q.BuildCountByRatePrefixSum(logger)
			if err != nil && logger != nil {
				logger.Warn("rebuild BuildCountByRatePrefixSum failed", zap.Error(err))
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
		histogram[c.Rate-minRate] = int64(c.Count)
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
	if rndStart >= rndEnd {
		return 0, fmt.Errorf("invalid range: start %d end %d", start, end)
	}
	rnd := rand.Int64N(rndEnd-rndStart) + rndStart
	idx := sort.Search(len(countByRatePrefixSum), func(i int) bool {
		return countByRatePrefixSum[i] > rnd
	})
	return idx - 1 + minCountRate, nil
}

func (q *Queries) GetPicByUserRate(ctx context.Context, rate int) (SavedPic, error) {
	rnd := int64(rand.Uint64())
	targetRate := pgtype.Int4{Int32: int32(rate), Valid: true}
	result, err := q.getNsfwPicByRateAndRandKey(ctx, targetRate, rnd)
	if errors.Is(err, pgx.ErrNoRows) {
		return q.getNsfwPicByRateFirst(ctx, targetRate)
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
		_, err := q.createOrUpdateNsfwPic(ctx,
			fileUid,
			fileId,
			int32(botRate),
			rnd,
		)

		// RandKey 冲突 → 重试
		var sErr *pgconn.PgError
		if errors.As(err, &sErr) && sErr.Code == pgerrcode.UniqueViolation {
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
	newRate32 := int32(newRate)
	rate, err := q.getNsfwPicRateByUserId(ctx, fileUid, userID)
	if errors.Is(err, pgx.ErrNoRows) {
		err = q.createNsfwPicUserRate(ctx, fileUid, userID, newRate32)
		return false, 0, err
	} else if err != nil {
		return false, 0, err
	}
	if rate == newRate32 {
		return true, int64(rate), nil
	}
	err = q.updateNsfwPicUserRate(ctx, newRate32, fileUid, userID)
	return true, int64(rate), err
}
