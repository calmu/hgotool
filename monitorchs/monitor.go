// Package monitorchs
//
// ----------------develop info----------------
//
//	@Author Calmu
//	@DateTime 2026-1-4 19:51
//
// --------------------------------------------
package monitorchs

import (
	"fmt"
	"github.com/calmu/hgotool/hlog"
	"go.uber.org/zap"
	"sync"
	"time"
)

const (
	MonitorDuration time.Duration = time.Minute
	ModBase                       = "base" // base模式，此时不再赋值默认值
)

type Options[T any] func(m *MonitorChs[T])

type MonitorChs[T any] struct {
	chs             map[string][]chan T
	quitCh          chan struct{}
	monitorDuration time.Duration
	hLog            hlog.HLoggerBase
	mod             string
	lock            sync.RWMutex
	once            sync.Once
}

// NewMonitorChs
//
//	@Description:
//	@param options ...Options
//	@return *MonitorChs
//
// ----------------develop info----------------
//
//	@Author:		Calmu
//	@DateTime:		2024-01-04 19:57:10
//
// --------------------------------------------
func NewMonitorChs[T any](options ...Options[T]) *MonitorChs[T] {
	m := &MonitorChs[T]{
		chs: make(map[string][]chan T), // 初始化chs map
	}

	for _, option := range options {
		option(m)
	}

	if m.mod == ModBase {
		return m
	}
	// 确保在所有选项应用后仍有默认值
	if m.hLog == nil {
		m.hLog = hlog.GlobalLoggers["default"]
	}
	if m.monitorDuration == 0 {
		m.monitorDuration = MonitorDuration
	}
	return m
}

func WithChs[T any](name string, chs []chan T) Options[T] {
	return func(m *MonitorChs[T]) {
		if m.chs == nil {
			m.chs = make(map[string][]chan T)
		}
		m.chs[name] = chs
	}
}

func WithCh[T any](name string, chs ...chan T) Options[T] {
	return func(m *MonitorChs[T]) {
		if m.chs == nil {
			m.chs = make(map[string][]chan T)
		}
		if m.chs[name] == nil {
			m.chs[name] = chs
		} else {
			m.chs[name] = append(m.chs[name], chs...)
		}
	}
}

func WithDuration[T any](duration time.Duration) Options[T] {
	return func(m *MonitorChs[T]) {
		m.monitorDuration = duration
	}
}

func WithLog[T any](hLog hlog.HLoggerBase) Options[T] {
	return func(m *MonitorChs[T]) {
		m.hLog = hLog
	}
}

func WithHLog[T any]() Options[T] {
	return func(m *MonitorChs[T]) {
		m.hLog = hlog.GlobalLoggers["default"]
	}
}

func WithModBase[T any]() Options[T] {
	return func(m *MonitorChs[T]) {
		m.mod = ModBase
	}
}

// Run
// 因为go1.25开始提供sync.WaitGroup Go(f func())来管理goroutine，所以这里使用hwaitgroup.WaitGroup临时代替sync.WaitGroup解锁Go函数
// 正确使用例子为:
// var wg hwaitgroup.WaitGroup
// chs := make([]chan string, 0, 10)
//
//	for i := 0; i < 10; i++ {
//		chs = append(chs, make(chan string, 100))
//	}
//
// monitor := NewMonitorChs(WithChs("test", chs[:4]), WithDuration[string](time.Second*5))
// wg.Go(monitor.Start)
// wg.Wait()
func (m *MonitorChs[T]) Run() {
	m.quitCh = make(chan struct{}, 1)
	ticker := time.NewTicker(m.monitorDuration)

	for {
		select {
		case <-ticker.C:
			fields := m.GetMonitorLog()

			// 确保hLog不为nil
			if fields != nil && m.hLog != nil {
				m.hLog.Warn("ch len monitor", fields...)
			}
		case <-m.quitCh:
			ticker.Stop()
			return
		}
	}
}

func (m *MonitorChs[T]) Stop() {
	m.once.Do(func() {
		if m.quitCh != nil {
			close(m.quitCh)
		}
	})
}

func (m *MonitorChs[T]) GetMonitorLog() []zap.Field {
	m.lock.RLock()
	defer m.lock.RUnlock()

	if m.chs == nil {
		return nil
	}
	ll := 0
	for _, chs := range m.chs {
		ll += len(chs)
	}
	if ll == 0 {
		return nil
	}
	fields := make([]zap.Field, 0, ll)
	for name, chs := range m.chs {
		l := len(chs)
		for i, ch := range chs {
			if l == 1 {
				fields = append(fields, zap.Any(fmt.Sprintf("%sch len", name), len(ch)))
			} else {
				fields = append(fields, zap.Any(fmt.Sprintf("%sch%v len", name, i), len(ch)))
			}
		}
	}
	return fields
}

// AddCh
//
//	@Description: 添加一个ch,注意，name的唯一性
//	@receiver: m *MonitorChs[T]
//	@receiver m
//	@param name string
//	@param ch chan T
//
// ----------------develop info----------------
//
//	@Author:		Calmu
//	@DateTime:		2026-01-29 16:37:25
//
// --------------------------------------------
func (m *MonitorChs[T]) AddCh(name string, ch chan T) {
	m.lock.Lock()
	defer m.lock.Unlock()

	if m.chs == nil {
		m.chs = make(map[string][]chan T)
	}
	if m.chs[name] == nil {
		m.chs[name] = make([]chan T, 0, 1)
		m.chs[name] = append(m.chs[name], ch)
	} else {
		m.chs[name][0] = ch
	}
}
