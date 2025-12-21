package lrusf

import (
	"strconv"
	"sync"
	"sync/atomic"
	"testing"
)

func TestCacheGetAddRemove(t *testing.T) {
	cache := NewCache[int, string](2, func(k int) string { return strconv.Itoa(k) }, nil)

	cache.Add(1, "a")
	if v, ok := cache.TryGet(1); !ok || v != "a" {
		t.Fatalf("TryGet(1) = (%q,%v), want (%q,true)", v, ok, "a")
	}

	cache.Add(1, "b")
	if v, ok := cache.TryGet(1); !ok || v != "b" {
		t.Fatalf("TryGet(1) after overwrite = (%q,%v), want (%q,true)", v, ok, "b")
	}

	cache.Remove(1)
	if _, ok := cache.TryGet(1); ok {
		t.Fatalf("TryGet(1) after Remove should miss")
	}

	var fetchCalls atomic.Int32
	cache.Add(2, "c")
	v, err := cache.Get(2, func() (string, error) {
		fetchCalls.Add(1)
		return "fetch", nil
	})
	if err != nil {
		t.Fatalf("Get(2) error: %v", err)
	}
	if v != "c" {
		t.Fatalf("Get(2) = %q, want %q", v, "c")
	}
	if fetchCalls.Load() != 0 {
		t.Fatalf("fetch should not be called on cache hit")
	}

	v, err = cache.Get(3, func() (string, error) {
		fetchCalls.Add(1)
		return "d", nil
	})
	if err != nil {
		t.Fatalf("Get(3) error: %v", err)
	}
	if v != "d" {
		t.Fatalf("Get(3) = %q, want %q", v, "d")
	}
	if fetchCalls.Load() != 1 {
		t.Fatalf("fetch should be called once on miss, got %d", fetchCalls.Load())
	}
}

func TestCacheSingleflight(t *testing.T) {
	cache := NewCache[int, string](2, func(k int) string { return strconv.Itoa(k) }, nil)

	var fetchCalls atomic.Int32
	fetchStarted := make(chan struct{})
	fetchContinue := make(chan struct{})
	var once sync.Once

	fetch := func() (string, error) {
		fetchCalls.Add(1)
		once.Do(func() { close(fetchStarted) })
		<-fetchContinue
		return "value", nil
	}

	const workers = 8
	var wg sync.WaitGroup
	results := make(chan string, workers)
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			v, err := cache.Get(1, fetch)
			if err != nil {
				t.Errorf("Get error: %v", err)
				return
			}
			results <- v
		}()
	}

	<-fetchStarted
	close(fetchContinue)
	wg.Wait()
	close(results)

	if fetchCalls.Load() != 1 {
		t.Fatalf("fetch should be called once, got %d", fetchCalls.Load())
	}
	for v := range results {
		if v != "value" {
			t.Fatalf("result = %q, want %q", v, "value")
		}
	}
}

func TestCacheEvictionLRU(t *testing.T) {
	var evictedKey int
	var evictedValue string
	var evictedCount atomic.Int32
	cache := NewCache[int, string](2, func(k int) string { return strconv.Itoa(k) }, func(k int, v string) {
		evictedKey = k
		evictedValue = v
		evictedCount.Add(1)
	})

	cache.Add(1, "a")
	cache.Add(2, "b")
	_, err := cache.Get(1, func() (string, error) {
		return "", nil
	})
	if err != nil {
		t.Fatalf("Get error: %v", err)
	}
	cache.Add(3, "c")

	if evictedCount.Load() != 1 {
		t.Fatalf("expected 1 eviction, got %d", evictedCount.Load())
	}
	if evictedKey != 2 || evictedValue != "b" {
		t.Fatalf("evicted (%d,%q), want (2,%q)", evictedKey, evictedValue, "b")
	}
	if _, ok := cache.TryGet(2); ok {
		t.Fatalf("expected key 2 to be evicted")
	}
	if v, ok := cache.TryGet(1); !ok || v != "a" {
		t.Fatalf("expected key 1 to remain")
	}
	if v, ok := cache.TryGet(3); !ok || v != "c" {
		t.Fatalf("expected key 3 to remain")
	}
}

func TestCacheRange(t *testing.T) {
	cache := NewCache[int, string](3, func(k int) string { return strconv.Itoa(k) }, nil)
	cache.Add(1, "a")
	cache.Add(2, "b")
	cache.Add(3, "c")

	got := make(map[int]string, 3)
	for k, v := range cache.Range() {
		got[k] = v
	}

	if len(got) != 3 {
		t.Fatalf("Range size = %d, want 3", len(got))
	}
	if got[1] != "a" || got[2] != "b" || got[3] != "c" {
		t.Fatalf("Range values = %#v, want all keys with correct values", got)
	}
}
