// Package counter
//
// ----------------develop info----------------
//
//	@Author Calmu
//	@DateTime 2025-4-14 17:09
//
// --------------------------------------------
package counter

import (
	"encoding/json"
	"github.com/calmu/hgotool/hticker"
	"sync/atomic"
	"time"
)

type OptionPro func(*CounterPro)

type CounterPro struct {
	counter
	Total        atomic.Uint64          `json:"-"`
	SubCounters  map[string]*CounterPro `json:"sub,omitempty"`
	secondTicker *hticker.Ticker
	minuteTicker *hticker.Ticker
	hourTicker   *hticker.Ticker
}

func WithSecondTickerDefault() OptionPro {
	return func(c *CounterPro) {
		c.secondTicker = hticker.NewTicker(time.Second, hticker.WithTickFunc(func() { c.Minute.Add(c.ResetSecond()) }))
	}
}

func WithMinuteTickerDefault() OptionPro {
	return func(c *CounterPro) {
		c.minuteTicker = hticker.NewTicker(time.Minute, hticker.WithTickFunc(func() { c.Hour.Add(c.ResetMinute()) }))
	}
}

func WithHourTickerDefault() OptionPro {
	return func(c *CounterPro) {
		c.hourTicker = hticker.NewTicker(time.Hour, hticker.WithTickFunc(func() { c.Total.Add(c.ResetHour()) }))
	}
}

func WithSecondTicker(ticker *hticker.Ticker) OptionPro {
	return func(c *CounterPro) {
		c.secondTicker = ticker
	}
}

func WithMinuteTicker(ticker *hticker.Ticker) OptionPro {
	return func(c *CounterPro) {
		c.minuteTicker = ticker
	}
}

func WithHourTicker(ticker *hticker.Ticker) OptionPro {
	return func(c *CounterPro) {
		c.hourTicker = ticker
	}
}

func NewCounterPro() *CounterPro {
	return &CounterPro{
		SubCounters: make(map[string]*CounterPro),
	}
}

// --- 子计数器管理（需要锁） ------------------------------------------

// Sub 获取或创建子计数器（类似 Prometheus Labels）
func (c *CounterPro) Sub(key string) *CounterPro {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.SubCounters == nil {
		c.SubCounters = make(map[string]*CounterPro)
	}
	if _, ok := c.SubCounters[key]; !ok {
		c.SubCounters[key] = NewCounterPro()
	}
	return c.SubCounters[key]
}

// RemoveSub 删除子计数器
func (c *CounterPro) RemoveSub(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.SubCounters, key)
}

// VisitSubs 遍历所有子计数器（读锁）
func (c *CounterPro) VisitSubs(fn func(key string, sub *CounterPro)) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	for k, v := range c.SubCounters {
		fn(k, v)
	}
}

// Stop 停止所有计数器
func (c *CounterPro) Stop() {
	if c.secondTicker != nil {
		c.secondTicker.Stop()
	}
	if c.minuteTicker != nil {
		c.minuteTicker.Stop()
	}
	if c.hourTicker != nil {
		c.hourTicker.Stop()
	}
}

// --- JSON 序列化支持 ------------------------------------------------

// MarshalJSON 实现 json.Marshaler，将 atomic.Uint64 输出为普通 uint64
func (c *CounterPro) MarshalJSON() ([]byte, error) {
	// 构建临时结构体，避免递归调用 MarshalJSON
	type Alias CounterPro
	return json.Marshal(&struct {
		Total  uint64 `json:"total,omitempty"`
		Second uint64 `json:"second,omitempty"`
		Minute uint64 `json:"minute,omitempty"`
		Hour   uint64 `json:"hour,omitempty"`
		*Alias
	}{
		Total:  c.Total.Load(),
		Second: c.LoadSecond(),
		Minute: c.LoadMinute(),
		Hour:   c.LoadHour(),
		Alias:  (*Alias)(c),
	})
}
