package q

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetRandomRangeByWeight(t *testing.T) {
	as := assert.New(t)
	data := []PicRateCounter{
		{-1, 0},
		{0, 100},
		{2, 100},
		{4, 100},
		{6, 300},
	}
	countByRatePrefixSum = buildPrefixSumFromSparse(data)
	as.Equal(-1, minCountRate)
	table := make([]int64, len(countByRatePrefixSum)-1)
	const totalCnt = 40000
	as.Equal(6-(-1)+1 /*包含0值*/ +1 /*包含二分搜索的起始值*/, len(countByRatePrefixSum))
	for range totalCnt {
		idx, err := getRandomRangeByWeight(minCountRate, len(countByRatePrefixSum)-1+minCountRate)
		as.Nil(err)
		as.True(idx < len(countByRatePrefixSum)-1+minCountRate)
		as.True(idx >= minCountRate)
		table[idx-minCountRate]++
	}
	probabilityTable := make([]float64, len(countByRatePrefixSum)-1)
	sum := 0
	for _, d := range data {
		sum += int(d.Count)
	}
	for _, d := range data {
		probabilityTable[int(d.Rate)-minCountRate] = float64(d.Count) / float64(sum)
	}
	for i := range table {
		delta := float64(table[i])/totalCnt - probabilityTable[i]
		as.Greater(delta, -0.05)
		as.Less(delta, 0.05)
	}

}
