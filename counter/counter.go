// Package counter
//
// ----------------develop info----------------
//
//	@Author Calmu
//	@DateTime 2025-4-13 16:51
//
// --------------------------------------------
package counter

import (
	"encoding/json"
	"sync"
	"sync/atomic"
)

type counter struct {
	Second atomic.Uint64 `json:"-"` // 自定义 JSON 处理
	Minute atomic.Uint64 `json:"-"`
	Hour   atomic.Uint64 `json:"-"`
	mu     sync.RWMutex  // 仅保护 SubCounters
}

// Counter 支持秒/分/时三级计数，原子操作，可嵌套子计数器
type Counter struct {
	counter
	SubCounters map[string]*Counter `json:"sub,omitempty"`
}

// NewCounter 创建一个新的计数器
func NewCounter() *Counter {
	return &Counter{
		SubCounters: make(map[string]*Counter),
	}
}

// --- 原子操作（无锁） -------------------------------------------------

// AddSecond 原子增加秒级计数
func (c *counter) AddSecond(v uint64) {
	c.Second.Add(v)
}

// AddMinute 原子增加分级计数
func (c *counter) AddMinute(v uint64) {
	c.Minute.Add(v)
}

// AddHour 原子增加小时级计数
func (c *counter) AddHour(v uint64) {
	c.Hour.Add(v)
}

// ResetSecond 原子重置秒计数并返回旧值
func (c *counter) ResetSecond() uint64 {
	return c.Second.Swap(0)
}

// ResetMinute 原子重置分计数并返回旧值
func (c *counter) ResetMinute() uint64 {
	return c.Minute.Swap(0)
}

// ResetHour 原子重置时计数并返回旧值
func (c *counter) ResetHour() uint64 {
	return c.Hour.Swap(0)
}

// LoadSecond 原子读取秒计数
func (c *counter) LoadSecond() uint64 {
	return c.Second.Load()
}

// LoadMinute 原子读取分计数
func (c *counter) LoadMinute() uint64 {
	return c.Minute.Load()
}

// LoadHour 原子读取时计数
func (c *counter) LoadHour() uint64 {
	return c.Hour.Load()
}

// Get 一次性获取三个计数器的当前值（非原子快照，但各自原子读取）
// 监控场景允许轻微不一致，若需要严格一致可加读锁（但会牺牲性能）
func (c *counter) Get() (second, minute, hour uint64) {
	return c.LoadSecond(), c.LoadMinute(), c.LoadHour()
}

// --- 子计数器管理（需要锁） ------------------------------------------

// Sub 获取或创建子计数器（类似 Prometheus Labels）
func (c *Counter) Sub(key string) *Counter {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.SubCounters == nil {
		c.SubCounters = make(map[string]*Counter)
	}
	if _, ok := c.SubCounters[key]; !ok {
		c.SubCounters[key] = NewCounter()
	}
	return c.SubCounters[key]
}

// RemoveSub 删除子计数器
func (c *Counter) RemoveSub(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.SubCounters, key)
}

// VisitSubs 遍历所有子计数器（读锁）
func (c *Counter) VisitSubs(fn func(key string, sub *Counter)) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	for k, v := range c.SubCounters {
		fn(k, v)
	}
}

// --- JSON 序列化支持 ------------------------------------------------

// MarshalJSON 实现 json.Marshaler，将 atomic.Uint64 输出为普通 uint64
func (c *Counter) MarshalJSON() ([]byte, error) {
	// 构建临时结构体，避免递归调用 MarshalJSON
	type Alias Counter
	return json.Marshal(&struct {
		Second uint64 `json:"second,omitempty"`
		Minute uint64 `json:"minute,omitempty"`
		Hour   uint64 `json:"hour,omitempty"`
		*Alias
	}{
		Second: c.LoadSecond(),
		Minute: c.LoadMinute(),
		Hour:   c.LoadHour(),
		Alias:  (*Alias)(c),
	})
}
