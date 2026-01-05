// Package monitorchs
//
// ----------------develop info----------------
//
//	@Author xunmuhuang@rastar.com
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
)

type Options[T any] func(m *MonitorChs[T])

type MonitorChs[T any] struct {
	chs             map[string][]chan T
	quitCh          chan struct{}
	monitorDuration time.Duration
	hLog            hlog.HLoggerBase
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

func (m *MonitorChs[T]) Run(wg *sync.WaitGroup) {
	m.quitCh = make(chan struct{}, 1)
	ticker := time.NewTicker(m.monitorDuration)
	go func() {
		defer wg.Done()
		for {
			select {
			case <-ticker.C:
				if m.chs == nil {
					continue
				}
				ll := 0
				for _, chs := range m.chs {
					ll += len(chs)
				}
				if ll == 0 {
					continue
				}
				fields := make([]zap.Field, 0, ll)
				for name, chs := range m.chs {
					for i, ch := range chs {
						fields = append(fields, zap.Any(fmt.Sprintf("%sch%v len", name, i), len(ch)))
					}
				}

				// 确保hLog不为nil
				if m.hLog != nil {
					m.hLog.Warn("ch len monitor", fields...)
				}
			case <-m.quitCh:
				ticker.Stop()
				return
			}
		}
	}()
}

func (m *MonitorChs[T]) Stop() {
	var once sync.Once
	once.Do(func() {
		if m.quitCh != nil {
			m.quitCh <- struct{}{}
			close(m.quitCh)
		}
	})
}
