// Package heartbeat
//
// ----------------develop info----------------
//
//	@Author Calmu
//	@DateTime 2025-2-24 15:06
//
// --------------------------------------------
package heartbeat

import (
	"github.com/calmu/hgotool/hticker"
	"sync"
	"time"
)

type Task struct {
	name          string
	initFunc      func()
	runFunc       func()
	stopFunc      func()
	heartbeat     time.Time // 心跳时间
	state         string    // task state, default is StateStart, can be StateStart, StatePause, StateStop
	isRunning     bool      // task is running, default is false
	stopTime      time.Time // task stop time
	lock          sync.RWMutex
	ticker        *hticker.Ticker
	duration      time.Duration
	runTickerFlag bool // 是否让框架逻辑启动Ticker，default is false
}

type TaskOption func(*Task)

func WithInitFunc(f func()) TaskOption {
	return func(tg *Task) {
		tg.initFunc = f
	}
}

func WithRunFunc(f func()) TaskOption {
	return func(tg *Task) {
		tg.runFunc = f
	}
}

func WithStopFunc(f func()) TaskOption {
	return func(tg *Task) {
		tg.stopFunc = f
	}
}

func WithRunTickerFlag(f bool) TaskOption {
	return func(tg *Task) {
		tg.runTickerFlag = f
	}
}

func WithTaskDuration(d time.Duration) TaskOption {
	return func(tg *Task) {
		tg.duration = d
	}
}

func NewTask(name string, options ...TaskOption) *Task {
	t := &Task{name: name}
	for _, option := range options {
		option(t)
	}
	return t
}

// Heartbeat 上报任务心跳，必须在任务函数中调用
func (t *Task) Heartbeat() {
	t.lock.Lock()
	defer t.lock.Unlock()

	t.heartbeat = time.Now()
}

// StartHeartbeat 启动任务心跳定时发送心跳，请在task的实体方法中调用(可以自己启动，也可以让框架启动)
func (t *Task) StartHeartbeat() {
	t.lock.Lock()
	defer t.lock.Unlock()

	if t.ticker != nil {
		t.ticker.Stop()
	}
	t.ticker = hticker.NewTicker(time.Second*10, hticker.WithGoroutine(true), hticker.WithTickFunc(t.Heartbeat))
	t.ticker.Start()
}

// StopHeartbeat 停止任务心跳定时发送心跳
func (t *Task) StopHeartbeat() {
	if t.ticker != nil {
		t.ticker.Stop()
	}
}

func (t *Task) stop(state string) {
	t.lock.Lock()
	t.state = state
	t.lock.Unlock()

	if t.stopFunc != nil {
		t.stopFunc()
	}
}
