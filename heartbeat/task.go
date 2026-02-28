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
	"sync"
	"time"
)

type Task struct {
	name      string
	initFunc  func()
	runFunc   func()
	stopFunc  func()
	heartbeat time.Time // 心跳时间
	state     string    // task state, default is StateStart, can be StateStart, StatePause, StateStop
	isRunning bool      // task is running, default is false
	stopTime  time.Time // task stop time
	lock      sync.RWMutex
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

func NewTask(name string, options ...TaskOption) *Task {
	t := &Task{name: name}
	for _, option := range options {
		option(t)
	}
	return t
}

// Heartbeat 上报任务心跳，必须在任务函数中调用，且必须传入父级的 heartbeat 心跳实例来加锁
func (t *Task) Heartbeat(hb *Heartbeat) {
	hb.lock.Lock()
	defer hb.lock.Unlock()

	t.heartbeat = time.Now()
}
