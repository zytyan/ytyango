package lrusf

import (
	"container/list"
	"iter"
	"sync"

	"golang.org/x/sync/singleflight"
)

type Cache[K comparable, V any] struct {
	mu      sync.Mutex
	cap     int
	ll      *list.List
	items   map[K]*list.Element
	sf      singleflight.Group
	keyFn   func(K) string
	onEvict func(key K, value V) // NEW
}

type entry[K comparable, V any] struct {
	key   K
	value V
}

// New 创建缓存。
// keyFn：将 K 映射为 singleflight 使用的 string key（必须稳定且唯一）。
// onEvict：容量驱逐时回调；可为 nil。
func New[K comparable, V any](cap int, keyFn func(K) string, onEvict func(K, V)) *Cache[K, V] {
	if cap <= 0 {
		panic("cap must be > 0")
	}
	if keyFn == nil {
		panic("keyFn must not be nil")
	}
	return &Cache[K, V]{
		cap:     cap,
		ll:      list.New(),
		items:   make(map[K]*list.Element, cap),
		keyFn:   keyFn,
		onEvict: onEvict,
	}
}

// Get
// 1. 命中缓存直接返回
// 2. 未命中使用 singleflight 合并并发 fetch
// 3. 成功才写入缓存
func (c *Cache[K, V]) Get(key K, fetch func() (V, error)) (V, error) {
	if v, ok := c.get(key); ok {
		return v, nil
	}

	vAny, err, _ := c.sf.Do(c.keyFn(key), func() (any, error) {
		if v, ok := c.get(key); ok {
			return v, nil
		}
		v, e := fetch()
		if e != nil {
			var zero V
			return zero, e
		}
		c.add(key, v)
		return v, nil
	})

	if err != nil {
		var zero V
		return zero, err
	}
	return vAny.(V), nil
}

func (c *Cache[K, V]) TryGet(key K) (V, bool) { return c.get(key) }
func (c *Cache[K, V]) Add(key K, value V)     { c.add(key, value) }

// Remove 删除指定 key（默认不触发 onEvict）
func (c *Cache[K, V]) Remove(key K) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if ele, ok := c.items[key]; ok {
		c.ll.Remove(ele)
		delete(c.items, key)
	}
}

func (c *Cache[K, V]) Len() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return len(c.items)
}

func (c *Cache[K, V]) Range() iter.Seq2[K, V] {
	return func(yield func(K, V) bool) {
		c.mu.Lock()
		defer c.mu.Unlock()
		for k, v := range c.items {
			if !yield(k, v.Value.(entry[K, V]).value) {
				return
			}
		}
	}
}

// --- internal ---

func (c *Cache[K, V]) get(key K) (V, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if ele, ok := c.items[key]; ok {
		c.ll.MoveToFront(ele)
		return ele.Value.(entry[K, V]).value, true
	}
	var zero V
	return zero, false
}

func (c *Cache[K, V]) add(key K, value V) {
	var (
		evictedKey   K
		evictedValue V
		evicted      bool
	)

	c.mu.Lock()

	if ele, ok := c.items[key]; ok {
		ele.Value = entry[K, V]{key: key, value: value}
		c.ll.MoveToFront(ele)
		c.mu.Unlock()
		return
	}

	ele := c.ll.PushFront(entry[K, V]{key: key, value: value})
	c.items[key] = ele

	if len(c.items) > c.cap {
		back := c.ll.Back()
		if back != nil {
			ent := back.Value.(entry[K, V])
			delete(c.items, ent.key)
			c.ll.Remove(back)

			if c.onEvict != nil {
				evictedKey, evictedValue, evicted = ent.key, ent.value, true
			}
		}
	}

	c.mu.Unlock()

	// 回调放在锁外，避免死锁/重入问题
	if evicted {
		c.onEvict(evictedKey, evictedValue)
	}
}
