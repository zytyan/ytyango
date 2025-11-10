package groupstatv2

import (
	"bytes"
	"encoding/gob"
	"slices"
	"sync/atomic"

	"github.com/puzpuzpuz/xsync/v3"
)

type Counter struct {
	atomic.Int64
}

func (c *Counter) Inc() {
	c.Add(1)
}

func (c *Counter) GobEncode() ([]byte, error) {
	v := c.Load()
	var buf bytes.Buffer
	if err := gob.NewEncoder(&buf).Encode(v); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func (c *Counter) GobDecode(data []byte) error {
	var v int64
	buf := bytes.NewBuffer(data)
	if err := gob.NewDecoder(buf).Decode(&v); err != nil {
		return err
	}
	c.Store(v)
	return nil
}

func (c *Counter) Value() any {
	return c.Load()
}

type EncodableMap[K comparable, V any] struct {
	*xsync.MapOf[K, V]
}

func (e *EncodableMap[K, V]) GobEncode() ([]byte, error) {
	buf := &bytes.Buffer{}
	tmpMap := xsync.ToPlainMapOf(e.MapOf)
	enc := gob.NewEncoder(buf)
	err := enc.Encode(tmpMap)
	return buf.Bytes(), err
}

func (e *EncodableMap[K, V]) GobDecode(data []byte) error {
	tmpMap := make(map[K]V)
	buf := bytes.NewBuffer(data)
	if err := gob.NewDecoder(buf).Decode(&tmpMap); err != nil {
		return err
	}
	e.MapOf = xsync.NewMapOf[K, V]()
	for k, v := range tmpMap {
		e.Store(k, v)
	}
	return nil
}

func (e *EncodableMap[K, V]) Value() any {
	return xsync.ToPlainMapOf(e.MapOf)
}

type MapPair[K comparable, V any] struct {
	Key   K
	Value V
}

func (e *EncodableMap[K, V]) SortedFunc(fn func(p1, p2 MapPair[K, V]) int) []MapPair[K, V] {
	sz := e.MapOf.Size()
	arr := make([]MapPair[K, V], 0, sz)
	e.MapOf.Range(func(key K, value V) bool {
		arr = append(arr, MapPair[K, V]{key, value})
		return true
	})
	slices.SortFunc(arr, fn)
	return arr
}

func NewEncodableMapOf[K comparable, V any](size int) *EncodableMap[K, V] {
	return &EncodableMap[K, V]{
		MapOf: xsync.NewMapOf[K, V](xsync.WithPresize(size)),
	}
}
