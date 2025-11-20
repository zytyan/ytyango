package q

import (
	"database/sql/driver"
	"fmt"
	"runtime"
	"time"
	"weak"

	"github.com/puzpuzpuz/xsync/v3"
)

type UnixTime struct {
	time.Time
}

func (u *UnixTime) Scan(value any) error {
	switch val := value.(type) {
	case int64:
		u.Time = time.Unix(val, 0)
		return nil
	case float64:
		u.Time = time.Unix(int64(val), 0)
		return nil
	default:
		return fmt.Errorf("cannot convert %v of type %T to UnixTime", value, value)
	}
}

func (u *UnixTime) Value() (driver.Value, error) {
	return u.Unix(), nil
}

type WeakMap[K comparable, V any] xsync.MapOf[K, weak.Pointer[V]]

func NewWeakMap[K comparable, V any]() *WeakMap[K, V] {
	m := xsync.NewMapOf[K, weak.Pointer[V]]()
	return (*WeakMap[K, V])(m)
}

func (w *WeakMap[K, V]) inner() *xsync.MapOf[K, weak.Pointer[V]] {
	return (*xsync.MapOf[K, weak.Pointer[V]])(w)
}

func (w *WeakMap[K, V]) Store(k K, v *V) {
	pv := weak.Make(v)
	runtime.AddCleanup(v, func(s *xsync.MapOf[K, weak.Pointer[V]]) {
		w.inner().Delete(k)
	}, w.inner())
	w.inner().Store(k, pv)
}

func (w *WeakMap[K, V]) LoadOrCompute(k K, compute func() *V) (*V, bool) {
	// 1. 尝试原子加载
	if pv, loaded := w.inner().Load(k); loaded {
		// 尝试解析 weak.Pointer
		if v := pv.Value(); v != nil {
			// 成功加载并有效
			return v, true
		} else {
			// 已经被垃圾回收，我们将尝试用计算的新值替换它。
			// 继续执行到 2. LoadOrComputeWithStorage 逻辑。
		}
	}

	// 2. 如果加载失败 (第一次或被回收后)，则计算新值并尝试存储。
	// 使用 xsync.MapOf 的 LoadOrCompute 确保只有一个 goroutine 负责计算和存储。

	// 我们需要一个函数来处理存储逻辑，包括计算新值和注册 Cleanup。
	// 这个函数只在 LoadOrStore 内部的 compute 逻辑被执行时调用。
	newPV, loaded := w.inner().LoadOrCompute(k, func() weak.Pointer[V] {
		// 这是 compute 函数，只在 key 不存在时执行

		// 计算新值 (可能耗时)
		v := compute()

		// 创建 Weak Pointer
		pv := weak.Make(v)

		// 注册 Cleanup 函数：当 v 被 GC 时，从 WeakMap 中删除键 k
		runtime.AddCleanup(v, func(s *xsync.MapOf[K, weak.Pointer[V]]) {
			s.Delete(k)
		}, w.inner())

		return pv
	})

	// 3. 检查最终的结果

	// 如果是 Loaded = true，表示 map 中原来就有值，我们尝试解析它
	if loaded {
		if v := newPV.Value(); v != nil {
			// 原来的值有效
			return v, true
		} else {
			// 原来的值无效 (已被 GC)。
			// 在 LoadOrCompute 的实现中，如果 key 存在，LoadOrCompute 不会重新执行 compute 函数。
			// 因此，我们仍返回 not found (nil, false)，并依赖 Store 中注册的 Cleanup 最终删除它。
			// 为了 LoadOrCompute 的语义完整性，可以再 Load 一次或直接返回未找到。
			// 这里我们选择返回未找到，并依赖 GC 的异步清理。
			// *注意：一个更复杂的实现可能会尝试 Delete/Store 循环来替换失效的值，
			// 但对于 WeakMap 来说，依赖 GC 清理是更简洁的策略。*
			w.inner().Delete(k) // 尝试立即清理，提高下次查询效率
			return nil, false
		}
	} else {
		// 如果是 Loaded = false，表示 compute 函数被执行，新值已存储。
		// 我们需要再次 Get 来获取实际值 (防止 compute 返回 nil)。
		if v := newPV.Value(); v != nil {
			return v, false // 返回新计算的值，loaded=false 表示是新计算的
		}
		// compute 返回了 nil，且已存储。返回未找到。
		return nil, false
	}
}

func (w *WeakMap[K, V]) Delete(k K) {
	// xsync.MapOf 的 Delete 足够了。
	w.inner().Delete(k)
	// 注意：Delete 只是从 map 中移除，但不会阻止 weak.Pointer 所指向的值被 GC，
	// 也不会取消 Store 中注册的 runtime.AddCleanup。
}

func (w *WeakMap[K, V]) Range(f func(k K, v *V) bool) {
	w.inner().Range(func(k K, pv weak.Pointer[V]) bool {
		// 尝试解析 weak.Pointer
		if v := pv.Value(); v != nil {
			// 如果值仍然有效，则调用回调函数
			return f(k, v)
		} else {
			// 如果值已被 GC，则从 Map 中删除并继续迭代。
			// 返回 true 以继续 Range。
			w.inner().Delete(k)
			return true
		}
	})
}

func (w *WeakMap[K, V]) Size() int {
	return w.inner().Size()
}

// Load 从 WeakMap 中加载键 k 对应的值。
// 如果值已被 GC，则返回 nil, false，并清理该键。
func (w *WeakMap[K, V]) Load(k K) (*V, bool) {
	if pv, ok := w.inner().Load(k); ok {
		if v := pv.Value(); v != nil {
			return v, true
		} else {
			// 值已被 GC，删除键 k。
			w.inner().Delete(k)
			return nil, false
		}
	}
	return nil, false
}