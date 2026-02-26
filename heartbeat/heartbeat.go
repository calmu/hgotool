// Package heartbeat
//
// ----------------develop info----------------
//
//	@Author Calmu
//	@DateTime 2025-2-24 15:05
//
// --------------------------------------------
package heartbeat

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/calmu/hgotool/hlog"
	"github.com/calmu/hgotool/hticker"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"sync"
	"time"
)

const (
	StateStop  = "stop"
	StateStart = "start"
	StatePause = "pause"
)

type Heartbeat struct {
	ctx               context.Context
	lock              sync.RWMutex
	ticker            *hticker.Ticker
	tgs               []*TaskGroup
	tgsMap            map[string]int
	tickerDuration    time.Duration
	saveHeartbeatFunc func(key string, val string)
	getHeartbeatFunc  func(key string) (string, error)
	cachePrefix       string
	logger            hlog.HLoggerBase
}

type Option func(*Heartbeat)

func WithTickerDuration(d time.Duration) Option {
	return func(hb *Heartbeat) {
		hb.tickerDuration = d
	}
}

func WithSaveHeartbeatFunc(f func(key string, val string)) Option {
	return func(hb *Heartbeat) {
		hb.saveHeartbeatFunc = f
	}
}

func WithGetHeartbeatFunc(f func(key string) (string, error)) Option {
	return func(hb *Heartbeat) {
		hb.getHeartbeatFunc = f
	}
}

func WithCachePrefix(prefix string) Option {
	return func(hb *Heartbeat) {
		hb.cachePrefix = prefix
	}
}

func WithLogger(logger hlog.HLoggerBase) Option {
	return func(hb *Heartbeat) {
		hb.logger = logger
	}
}

func NewHeartbeat(ctx context.Context, options ...Option) *Heartbeat {
	hb := &Heartbeat{
		ctx:            ctx,
		tgs:            make([]*TaskGroup, 0, 100),
		tgsMap:         make(map[string]int, 100),
		tickerDuration: time.Second * 30,
		cachePrefix:    "heartbeat:",
	}
	for _, option := range options {
		option(hb)
	}

	return hb
}

func (hb *Heartbeat) Add(tg *TaskGroup) {
	hb.lock.Lock()
	defer hb.lock.Unlock()
	hb.tgs = append(hb.tgs, tg)
	hb.tgsMap[tg.group] = len(hb.tgs) - 1
}

func (hb *Heartbeat) Start(wg *sync.WaitGroup) {
	defer wg.Done()

	hbList := make(map[string]*HeartbeatInfo, len(hb.tgs)*10)
	lock := sync.RWMutex{}

	var wgA sync.WaitGroup

	hb.runGroup(&wgA)
	syncRunFunc := func() {
		hb.collect(hbList)
		hb.syncHeartbeatToCache(hbList)
	}
	defer syncRunFunc()
	syncRunFunc()

	// 监听ctx
	go func() {
		<-hb.ctx.Done()
		hb.Stop()
	}()

	// 等待所有任务组运行,并记录运行状态
	defer func() {
		wgA.Wait()
		for _, tg := range hb.tgs {
			for _, task := range tg.tasks {
				task.stopTime = time.Now()
				if task.state == StateStart {
					task.state = StateStop
				}
			}
		}
	}()

	// 定时读取状态控制启停
	readTicker := hticker.NewTicker(hb.tickerDuration, hticker.WithTickFunc(func() {
		if hb.logger != nil {
			hb.logger.Info("readTicker tick", zap.Any("hbList", hbList))
		}
		hb.runGroup(&wgA)
	}))
	defer readTicker.Stop()
	readTicker.Start()

	// 定时保存心跳
	saveTicker := hticker.NewTicker(hb.tickerDuration, hticker.WithTickFunc(func() {
		lock.RLock()
		defer lock.RUnlock()

		hb.syncHeartbeatToCache(hbList)
	}))
	defer saveTicker.Stop()
	saveTicker.Start()

	// 定时监测任务组并收集心跳
	hb.ticker = hticker.NewTicker(hb.tickerDuration, hticker.WithRunFirst(true), hticker.WithGoroutine(false), hticker.WithTickFunc(func() {
		hb.lock.RLock()
		defer hb.lock.RUnlock()

		lock.Lock()
		defer lock.Unlock()

		hb.collect(hbList)
	}))
	hb.ticker.Start()

	if hb.logger != nil {
		hb.logger.Info("hb ticker stop return", zap.Any("hbList", hbList))
	}

}

func (hb *Heartbeat) syncHeartbeatToCache(hbList map[string]*HeartbeatInfo) {
	if hb.logger != nil {
		hb.logger.Info("saveTicker tick", zap.Any("hbList", hbList))
	}

	for key, val := range hbList {
		valStr, _ := json.Marshal(val)
		hb.saveHeartbeatFunc(key, string(valStr))
	}
}

func (hb *Heartbeat) collect(hbList map[string]*HeartbeatInfo) {
	if hb.logger != nil {
		hb.logger.Info("hb.ticker tick", zap.Any("hbList", hbList))
	}
	for _, tg := range hb.tgs {
		for _, task := range tg.tasks {
			key := tg.buildTaskInfoKey(task)
			if _, ok := hbList[key]; !ok {
				hbList[key] = &HeartbeatInfo{
					Key:       key,
					State:     task.state,
					CreatedAt: time.Now(),
				}
			}
			if task.isRunning {
				hbList[key].State = StateStart
			} else if task.state == StateStop {
				hbList[key].State = StateStop
				hbList[key].StopTime = task.stopTime
			} else if task.state == StatePause {
				hbList[key].State = StatePause
				hbList[key].StopTime = task.stopTime
			}

			// Update heartbeat if state is not Stop
			if hbList[key].State != StateStop {
				hbList[key].Heartbeat = task.heartbeat // Update heartbeat
			}
			hbList[key].UpdatedAt = time.Now()
		}
	}
}

func (hb *Heartbeat) runGroup(wg *sync.WaitGroup) {
	hb.lock.Lock()
	defer hb.lock.Unlock()

	for _, tg := range hb.tgs {
		// 读组状态
		gKey := tg.buildStateKey()

		if gStr, err := hb.getHeartbeatFunc(gKey); err != nil && !errors.Is(err, redis.Nil) {
			if hb.logger != nil {
				hb.logger.Error("getHeartbeatFunc error", zap.Error(err), zap.String("gKey", gKey))
			}
			continue
		} else {
			// 同时应该初始化组状态到外部缓存
			if gStr == "" {
				gStr = StateStart
				hb.saveHeartbeatFunc(gKey, gStr)
			}
			// 操作组
			switch gStr {
			case StateStart:
				tg.state = StateStart
				// 读取task的外部状态
				tg.syncTasksStateFromHeartbeat(hb)
				tg.run(wg)
			case StatePause, StateStop:
				tg.state = gStr
				tg.stop()
				continue
			}
		}
	}
}

func (hb *Heartbeat) Stop() {
	hb.ticker.Stop()
}

// HeartbeatInfo task heartbeat info
type HeartbeatInfo struct {
	Key       string    `json:"-"`
	State     string    `json:"state,omitempty"`     // task state, 代表的是当前task状态，非总控的状态
	Heartbeat time.Time `json:"heartbeat,omitempty"` // task heartbeat
	StopTime  time.Time `json:"stop_time,omitempty"` // task stop time
	CreatedAt time.Time `json:"created_at,omitempty"`
	UpdatedAt time.Time `json:"updated_at,omitempty"`
}
