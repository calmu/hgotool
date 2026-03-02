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
		tickerDuration: time.Second * 10,
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

type HbInfoList struct {
	List map[string]*HeartbeatInfo `json:"list"`
	Lock sync.RWMutex
}

func NewHbInfoList(l int) *HbInfoList {
	return &HbInfoList{
		List: make(map[string]*HeartbeatInfo, l),
	}
}

func (hb *Heartbeat) Start(wg *sync.WaitGroup) {
	defer wg.Done()

	hbList := NewHbInfoList(len(hb.tgs) * 10)

	var wgA sync.WaitGroup

	// 先运行一次
	hb.runGroup(&wgA)
	hb.collect(hbList)
	hb.syncHeartbeatToCache(hbList)

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
		hb.collect(hbList)
		hb.syncHeartbeatToCache(hbList)
		if hb.logger != nil {
			hb.logger.Info("hb ticker stop return", zap.Any("hbList", hbList))
		}
	}()

	var wgB, wgC, wgD sync.WaitGroup

	wgB.Add(1)
	// 定时读取状态控制启停
	readTicker := hticker.NewTicker(hb.tickerDuration, hticker.WithTickFunc(func() {
		if hb.logger != nil {
			hbList.Lock.RLock()
			hbInfo, _ := json.Marshal(hbList)
			hbList.Lock.RUnlock()
			hb.logger.Info("readTicker tick", zap.String("hbList", string(hbInfo)))
		}
		hb.runGroup(&wgA)
	}), hticker.WithDeferFunc(func() {
		wgB.Done()
	}))
	readTicker.Start()

	wgC.Add(1)
	// 定时同步心跳到外部缓存
	saveTicker := hticker.NewTicker(hb.tickerDuration, hticker.WithTickFunc(func() {
		hbList.Lock.RLock()
		defer hbList.Lock.RUnlock()

		hb.syncHeartbeatToCache(hbList)
	}), hticker.WithDeferFunc(func() {
		wgC.Done()
	}))
	saveTicker.Start()

	wgD.Add(1)
	// 定时监测任务组并收集心跳
	hb.ticker = hticker.NewTicker(hb.tickerDuration, hticker.WithRunFirst(true), hticker.WithTickFunc(func() {
		hb.lock.RLock()
		defer hb.lock.RUnlock()

		hbList.Lock.Lock()
		defer hbList.Lock.Unlock()

		hb.collect(hbList)
	}), hticker.WithDeferFunc(func() {
		wgD.Done()
	}))
	hb.ticker.Start()

	// 监听ctx
	<-hb.ctx.Done()

	readTicker.Stop()
	wgB.Wait()

	saveTicker.Stop()
	wgC.Wait()

	hb.ticker.Stop()
	wgD.Wait()
}

func (hb *Heartbeat) buildCacheKey(key string) string {
	return hb.cachePrefix + key
}

func (hb *Heartbeat) syncHeartbeatToCache(hbList *HbInfoList) {
	if hb.logger != nil {
		hbInfo, _ := json.Marshal(hbList.List)
		hb.logger.Info("saveTicker tick", zap.ByteString("hbList", hbInfo))
	}

	for key, val := range hbList.List {
		valStr, _ := json.Marshal(val)
		hb.saveHeartbeatFunc(hb.buildCacheKey(key), string(valStr)) // Fixed: Added missing argument
	}
}

func (hb *Heartbeat) collect(hbList *HbInfoList) {
	if hb.logger != nil {
		hbInfo, _ := json.Marshal(hbList.List)
		hb.logger.Info("hb.ticker tick", zap.ByteString("hbList", hbInfo))
	}
	for _, tg := range hb.tgs {
		for _, task := range tg.tasks {
			func() {
				task.lock.RLock()
				defer task.lock.RUnlock()

				key := tg.buildTaskInfoKey(task)
				if _, ok := hbList.List[key]; !ok {
					hbList.List[key] = &HeartbeatInfo{
						Key:       key,
						State:     task.state,
						CreatedAt: time.Now(),
					}
				}
				if task.isRunning {
					hbList.List[key].State = StateStart
				} else if task.state == StateStop {
					hbList.List[key].State = StateStop
					hbList.List[key].StopTime = task.stopTime
				} else if task.state == StatePause {
					hbList.List[key].State = StatePause
					hbList.List[key].StopTime = task.stopTime
				}

				if task.heartbeat.After(hbList.List[key].CreatedAt.Add(-time.Second * 1)) {
					hbList.List[key].Heartbeat = task.heartbeat // Update heartbeat
				}
				hbList.List[key].UpdatedAt = time.Now()
			}()
		}
	}
}

func (hb *Heartbeat) runGroup(wg *sync.WaitGroup) {
	hb.lock.Lock()
	defer hb.lock.Unlock()

	for _, tg := range hb.tgs {
		// 读组状态
		gKey := tg.buildStateKey()

		if gStr, err := hb.getHeartbeatFunc(hb.buildCacheKey(gKey)); err != nil && !errors.Is(err, redis.Nil) { // Fixed: Added missing argument
			if hb.logger != nil {
				hb.logger.Error("getHeartbeatFunc error", zap.Error(err), zap.String("gKey", gKey))
			}
			continue
		} else {
			// 同时应该初始化组状态到外部缓存
			if gStr == "" {
				gStr = StateStart
				hb.saveHeartbeatFunc(hb.buildCacheKey(gKey), gStr) // Fixed: Added missing argument
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
				tg.stop(gStr)
				continue
			}
		}
	}
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
