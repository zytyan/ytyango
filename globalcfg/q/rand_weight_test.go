package q

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetRandomRangeByWeight(t *testing.T) {
	as := assert.New(t)
	data := []PicRateCounter{
		{0, 100},
		{2, 100},
		{4, 100},
		{6, 300},
	}
	countByRatePrefixSum = buildPrefixSumFromSparse(data)
	table := make([]int64, picRateCount)
	const totalCnt = 40000
	as.Equal(picRateCount+1, len(countByRatePrefixSum))
	for range totalCnt {
		idx, err := getRandomRangeByWeight(0, 7)
		as.Nil(err)
		as.True(idx < 7)
		as.True(idx >= 0)
		table[idx]++
	}
	probabilityTable := make([]float64, picRateCount)
	sum := 0
	for _, d := range data {
		if d.Rate < 7 {
			sum += int(d.Count)
		}
	}
	for _, d := range data {
		if d.Rate < 7 {
			probabilityTable[int(d.Rate)] = float64(d.Count) / float64(sum)
		}
	}
	for i := range table {
		delta := float64(table[i])/totalCnt - probabilityTable[i]
		as.Greater(delta, -0.05)
		as.Less(delta, 0.05)
	}

}

func TestGetRandomRangeByWeightRange(t *testing.T) {
	as := assert.New(t)
	countByRatePrefixSum = buildPrefixSumFromSparse([]PicRateCounter{
		{0, 100},
		{1, 100},
		{2, 100},
		{3, 100},
	})

	for range 1000 {
		idx, err := getRandomRangeByWeight(1, 3)
		as.Nil(err)
		as.True(idx >= 1)
		as.True(idx < 3)
	}
	_, err := getRandomRangeByWeight(3, 3)
	as.Error(err)
	_, err = getRandomRangeByWeight(-1, 3)
	as.Error(err)
	_, err = getRandomRangeByWeight(0, 9)
	as.Error(err)
}
