package q

import (
	"cmp"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math/rand"
	"slices"
	"sort"
	"sync"
	"time"

	"github.com/mattn/go-sqlite3"
)

var psMu sync.RWMutex

// prefix sum: ps[i] = rate=(minRate+i) 的累计数量
var countByRatePrefixSum []int64
var minCountRate int64 // prefixSum 的偏移量

func (q *Queries) InitCountByRatePrefixSum() error {
	psMu.Lock()
	defer psMu.Unlock()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// 必须保证 SQL 已经 ORDER BY rate；如果没保证这里再排一次
	counts, err := q.getPicRateCounts(ctx)
	if err != nil {
		return err
	}
	if len(counts) == 0 {
		countByRatePrefixSum = nil
		minCountRate = 0
		return nil
	}
	slices.SortFunc(counts, func(a, b PicRateCounter) int {
		return cmp.Compare(a.Rate, b.Rate)
	})

	minRate := counts[0].Rate
	maxRate := counts[len(counts)-1].Rate
	size := maxRate - minRate + 1

	ps := make([]int64, size)

	// histogram
	for _, c := range counts {
		ps[c.Rate-minRate] = c.Count
	}

	// prefix sum
	for i := int64(1); i < size; i++ {
		ps[i] += ps[i-1]
	}

	countByRatePrefixSum = ps
	minCountRate = minRate
	return nil
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
	// lazy init prefix sum to support already-populated databases
	psMu.RLock()
	needInit := countByRatePrefixSum == nil
	psMu.RUnlock()
	if needInit {
		if err = q.InitCountByRatePrefixSum(); err != nil {
			return
		}
	}

	psMu.RLock()
	defer psMu.RUnlock()
	// 无 prefixSum，或者 start/end 超界，或者 start > end
	if len(countByRatePrefixSum) == 0 || start > end {
		err = sql.ErrNoRows
		return
	}

	// clamp 范围到 prefix 可表示区间
	actualMin := minCountRate
	actualMax := minCountRate + int64(len(countByRatePrefixSum)-1)

	s := int64(start)
	e := int64(end)

	if e < actualMin || s > actualMax {
		err = sql.ErrNoRows
		return
	}

	if s < actualMin {
		s = actualMin
	}
	if e > actualMax {
		e = actualMax
	}
	if s > e {
		err = sql.ErrNoRows
		return
	}

	// prefixSum 下标区间
	startIdx := s - actualMin
	endIdx := e - actualMin

	// 区间内总数 = ps[end] - ps[start-1]
	var total int64
	if startIdx == 0 {
		total = countByRatePrefixSum[endIdx]
	} else {
		total = countByRatePrefixSum[endIdx] - countByRatePrefixSum[startIdx-1]
	}
	if total <= 0 {
		err = sql.ErrNoRows
		return
	}

	// 随机选择区间权重下的一个数
	rnd := rand.Int63n(total)

	// 转成全局 prefixSum 的目标值
	var target int64
	if startIdx == 0 {
		target = rnd
	} else {
		target = countByRatePrefixSum[startIdx-1] + rnd
	}

	// 二分定位真实 rate 下标
	idx := sort.Search(len(countByRatePrefixSum), func(i int) bool {
		return countByRatePrefixSum[i] > target
	})
	rate := actualMin + int64(idx)

	// 随机选一个该 rate 的图片
	rndKey := int64(rand.Uint64())
	result, err := q.getPicByRateAndRandKey(ctx, rate, rndKey)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		err = sql.ErrNoRows
		return
	}
	if result.FileID != "" {
		return result, nil
	}
	return q.getPicByRateFirst(ctx, rate)
}

func (q *Queries) AddPic(ctx context.Context, fileUid, fileId string, botRate int) error {
	psMu.Lock()
	defer psMu.Unlock()
	for i := 0; i < 3; i++ {
		rnd := int64(rand.Uint64())
		inserted, err := q.insertPic(ctx, insertPicParams{
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

		// === 以下是 prefix_sum 更新逻辑 ===

		userRate := inserted.UserRate

		// 未初始化 → 全量初始化
		if countByRatePrefixSum == nil {
			return q.InitCountByRatePrefixSum()
		}

		minRate := minCountRate
		maxRate := minRate + int64(len(countByRatePrefixSum)-1)

		// 越界 → 重新 Init
		if userRate < minRate || userRate > maxRate {
			return q.InitCountByRatePrefixSum()
		}

		// 在范围内 → 局部 prefix 更新
		idx := userRate - minRate
		for i := idx; i < int64(len(countByRatePrefixSum)); i++ {
			countByRatePrefixSum[i]++
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
