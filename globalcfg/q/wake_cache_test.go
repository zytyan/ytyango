package q

import (
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
)

type Small struct {
	i int
}
type Big struct {
	a int
	b string
	c []byte
	d bool
	f map[int]string
}

func addSmallCache(cache *WeakMap[int, Small]) {
	s := &Small{i: 1}
	cache.Store(100, s)
}

func addBigCache(cache *WeakMap[int, Big]) {
	b := &Big{a: 1, b: "bigCache", c: []byte("bytes"), f: make(map[int]string)}
	cache.Store(200, b)
}

func TestCache(t *testing.T) {
	as := assert.New(t)
	smallCache := NewWeakMap[int, Small]()
	addSmallCache(smallCache)
	s, ok := smallCache.Load(100)
	as.True(ok)
	as.NotNil(s)
	runtime.GC()
	s, ok = smallCache.Load(100)
	as.False(ok)
	as.Nil(s)

	bigCache := NewWeakMap[int, Big]()
	addBigCache(bigCache)
	b, ok := bigCache.Load(200)
	as.True(ok)
	as.NotNil(b)
	runtime.GC()
	b, ok = bigCache.Load(200)
	as.False(ok)
	as.Nil(b)
}
