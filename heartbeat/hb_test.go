// Package heartbeat
//
// ----------------develop info----------------
//
//	@Author Calmu
//	@DateTime 2025-2-25 11:15
//
// --------------------------------------------
package heartbeat

import (
	"encoding/json"
	"fmt"
	"github.com/calmu/hgotool/hlog"
	"github.com/calmu/hgotool/hsignal"
	"github.com/calmu/hgotool/hticker"
	"github.com/redis/go-redis/v9"
	"strings"
	"sync"
	"testing"
	"time"
)

var cacheList map[string]string

var lock sync.RWMutex

func init() {
	cacheList = make(map[string]string)
}

func getHeartbeat(key string) (string, error) {
	lock.RLock()
	defer lock.RUnlock()
	if val, ok := cacheList[key]; ok {
		return val, nil
	}
	return "", redis.Nil // Key not found or other error occurred
}

func setHeartbeat(key string, val string) {
	lock.Lock()
	defer lock.Unlock()
	cacheList[key] = val
}

func TestHbCtxStop(t *testing.T) {
	testHbStop(t, false)

	for key, s2 := range cacheList {
		var info HeartbeatInfo
		_ = json.Unmarshal([]byte(s2), &info)
		if strings.HasSuffix(key, "info") {
			if info.State != StateStop {
				t.Errorf("heartbeat state is not stop, %s", s2)
			}
			minTime := time.Now().AddDate(-1, 0, 0)
			if info.StopTime.Before(minTime) {
				t.Errorf("heartbeat stop time is before min time, %s", s2)
			}
			if info.Heartbeat.Before(minTime) {
				t.Errorf("heartbeat last heartbeat time is before min time, %s", s2)
			}
			if info.CreatedAt.Before(minTime) {
				t.Errorf("heartbeat created time is before min time, %s", s2)
			}
		}
	}
}

func TestHbRunStop(t *testing.T) {
	testHbStop(t, true)

	for key, s2 := range cacheList {
		var info HeartbeatInfo
		_ = json.Unmarshal([]byte(s2), &info)
		if strings.HasSuffix(key, "info") {
			if info.State != StateStop {
				t.Errorf("heartbeat state is not stop, %s", s2)
			}
			minTime := time.Now().AddDate(-1, 0, 0)
			if info.StopTime.Before(minTime) {
				t.Errorf("heartbeat stop time is before min time, %s", s2)
			}
			if info.Heartbeat.Before(minTime) {
				t.Errorf("heartbeat last heartbeat time is before min time, %s", s2)
			}
			if info.CreatedAt.Before(minTime) {
				t.Errorf("heartbeat created time is before min time, %s", s2)
			}
		}
	}
}

func testHbStop(t *testing.T, runStop bool) {
	ctx, _ := hsignal.ContextSignal(hlog.GetLogger("heartbeat").Warn)

	hb := NewHeartbeat(ctx,
		WithTickerDuration(time.Second*5),
		WithCachePrefix("heartbeat:"),
		WithGetHeartbeatFunc(getHeartbeat),
		WithSaveHeartbeatFunc(setHeartbeat),
		WithLogger(hlog.GetLogger("heartbeat")),
	)

	var task1 *Task

	task1 = NewTask("task-1", WithInitFunc(func() { t.Log("task-1 init") }), WithRunFunc(func() {
		defer t.Log("task-1 run end")
		t.Log("task-1 run")
		time.Sleep(time.Second * 10)
		t.Log(task1, time.Now())
		task1.Heartbeat(hb)
		// 如果是真运行结束，必须在这里执行心跳stop,避免重复拉起；
		if runStop {
			hb.Stop()
		}
	}))
	fmt.Println(task1, 2)

	tg := NewTaskGroup(WithGroupName("test-one1"), WithTaskList(task1))

	hb.Add(tg)

	var wg sync.WaitGroup
	wg.Add(1)
	go hb.Start(&wg)

	ticker := hticker.NewTicker(time.Second*5, hticker.WithTickFunc(func() {
		lock.RLock()
		defer lock.RUnlock()

		t.Log("cache", cacheList)
	}))
	ticker.Start()
	wg.Wait()
	ticker.Stop()
	t.Log("cache", cacheList)
}

func TestHbStateStop(t *testing.T) {
	cacheKeyPrefix := "test-one1:task-1:"
	stateCacheKey := fmt.Sprintf("%sstate", cacheKeyPrefix)
	setHeartbeat(stateCacheKey, StateStop)
	testHbStop(t, false)
	for key, s2 := range cacheList {
		if key == stateCacheKey {
			if s2 != StateStop {
				t.Errorf("heartbeat state is not stop, %s", s2)
			}
		} else if strings.HasPrefix(key, cacheKeyPrefix) {
			var info HeartbeatInfo
			_ = json.Unmarshal([]byte(s2), &info)
			if strings.HasSuffix(key, "info") {
				if info.State != StateStop {
					t.Errorf("heartbeat state is not stop, %s", s2)
				}
				minTime := time.Now().AddDate(-1, 0, 0)
				/*if info.StopTime.Before(minTime) {
					t.Errorf("heartbeat stop time is before min time, %s", s2)
				}
				if info.Heartbeat.Before(minTime) {
					t.Errorf("heartbeat last heartbeat time is before min time, %s", s2)
				}*/
				if info.CreatedAt.Before(minTime) {
					t.Errorf("heartbeat created time is before min time, %s", s2)
				}
			}
		}
	}
}

func TestHbRunAfterStateStop(t *testing.T) {
	cacheKeyPrefix := "test-one1:task-1:"
	stateCacheKey := fmt.Sprintf("%sstate", cacheKeyPrefix)
	go func() {
		time.Sleep(time.Second * 5)
		setHeartbeat(stateCacheKey, StateStop)
	}()
	testHbStop(t, false)
	for key, s2 := range cacheList {
		if key == stateCacheKey {
			if s2 != StateStop {
				t.Errorf("heartbeat state is not stop, %s", s2)
			}
		} else if strings.HasPrefix(key, cacheKeyPrefix) {
			var info HeartbeatInfo
			_ = json.Unmarshal([]byte(s2), &info)
			if strings.HasSuffix(key, "info") {
				if info.State != StateStop {
					t.Errorf("heartbeat state is not stop, %s", s2)
				}
				minTime := time.Now().AddDate(-1, 0, 0)
				if info.StopTime.Before(minTime) {
					t.Errorf("heartbeat stop time is before min time, %s", s2)
				}
				if info.Heartbeat.Before(minTime) {
					t.Errorf("heartbeat last heartbeat time is before min time, %s", s2)
				}
				if info.CreatedAt.Before(minTime) {
					t.Errorf("heartbeat created time is before min time, %s", s2)
				}
			}
		}
	}
}

func TestHbStatePause(t *testing.T) {
	cacheKeyPrefix := "test-one1:task-1:"
	stateCacheKey := fmt.Sprintf("%sstate", cacheKeyPrefix)
	setHeartbeat(stateCacheKey, StatePause)
	testHbStop(t, false)
	for key, s2 := range cacheList {
		if key == stateCacheKey {
			if s2 != StatePause {
				t.Errorf("heartbeat state is not pause, %s", s2)
			}
		} else if strings.HasPrefix(key, cacheKeyPrefix) {
			var info HeartbeatInfo
			_ = json.Unmarshal([]byte(s2), &info)
			if strings.HasSuffix(key, "info") {
				if info.State != StatePause {
					t.Errorf("heartbeat state is not pause, %s", s2)
				}
				minTime := time.Now().AddDate(-1, 0, 0)
				/*if info.StopTime.Before(minTime) {
					t.Errorf("heartbeat stop time is before min time, %s", s2)
				}
				if info.Heartbeat.Before(minTime) {
					t.Errorf("heartbeat last heartbeat time is before min time, %s", s2)
				}*/
				if info.CreatedAt.Before(minTime) {
					t.Errorf("heartbeat created time is before min time, %s", s2)
				}
			}
		}
	}
}

func TestHbRunAfterStatePause(t *testing.T) {
	cacheKeyPrefix := "test-one1:task-1:"
	stateCacheKey := fmt.Sprintf("%sstate", cacheKeyPrefix)
	go func() {
		time.Sleep(time.Second * 5)
		setHeartbeat(stateCacheKey, StatePause)
	}()
	testHbStop(t, false)
	for key, s2 := range cacheList {
		if key == stateCacheKey {
			if s2 != StatePause {
				t.Errorf("heartbeat state is not pause, %s", s2)
			}
		} else if strings.HasPrefix(key, cacheKeyPrefix) {
			var info HeartbeatInfo
			_ = json.Unmarshal([]byte(s2), &info)
			if strings.HasSuffix(key, "info") {
				if info.State != StatePause {
					t.Errorf("heartbeat state is not pause, %s", s2)
				}
				minTime := time.Now().AddDate(-1, 0, 0)
				if info.StopTime.Before(minTime) {
					t.Errorf("heartbeat stop time is before min time, %s", s2)
				}
				if info.Heartbeat.Before(minTime) {
					t.Errorf("heartbeat last heartbeat time is before min time, %s", s2)
				}
				if info.CreatedAt.Before(minTime) {
					t.Errorf("heartbeat created time is before min time, %s", s2)
				}
			}
		}
	}
}

func TestHbRunStartAfterPause(t *testing.T) {
	cacheKeyPrefix := "test-one1:task-1:"
	stateCacheKey := fmt.Sprintf("%sstate", cacheKeyPrefix)
	setHeartbeat(stateCacheKey, StatePause)
	go func() {
		time.Sleep(time.Second * 5)
		setHeartbeat(stateCacheKey, StateStart)
	}()
	testHbStop(t, true)
	for key, s2 := range cacheList {
		if key == stateCacheKey {
			if s2 != StateStart {
				t.Errorf("heartbeat state is not start, %s", s2)
			}
		} else if strings.HasPrefix(key, cacheKeyPrefix) {
			var info HeartbeatInfo
			_ = json.Unmarshal([]byte(s2), &info)
			if strings.HasSuffix(key, "info") {
				if info.State != StateStop {
					t.Errorf("heartbeat state is not stop, %s", s2)
				}
				minTime := time.Now().AddDate(-1, 0, 0)
				if info.StopTime.Before(minTime) {
					t.Errorf("heartbeat stop time is before min time, %s", s2)
				}
				if info.Heartbeat.Before(minTime) {
					t.Errorf("heartbeat last heartbeat time is before min time, %s", s2)
				}
				if info.CreatedAt.Before(minTime) {
					t.Errorf("heartbeat created time is before min time, %s", s2)
				}
			}
		}
	}
}

func TestHbRunStartAfterStop(t *testing.T) {
	cacheKeyPrefix := "test-one1:task-1:"
	stateCacheKey := fmt.Sprintf("%sstate", cacheKeyPrefix)
	setHeartbeat(stateCacheKey, StateStop)
	go func() {
		time.Sleep(time.Second * 5)
		setHeartbeat(stateCacheKey, StateStart)
	}()
	testHbStop(t, true)
	for key, s2 := range cacheList {
		if key == stateCacheKey {
			if s2 != StateStart {
				t.Errorf("heartbeat state is not start, %s", s2)
			}
		} else if strings.HasPrefix(key, cacheKeyPrefix) {
			var info HeartbeatInfo
			_ = json.Unmarshal([]byte(s2), &info)
			if strings.HasSuffix(key, "info") {
				if info.State != StateStop {
					t.Errorf("heartbeat state is not stop, %s", s2)
				}
				minTime := time.Now().AddDate(-1, 0, 0)
				if info.StopTime.Before(minTime) {
					t.Errorf("heartbeat stop time is before min time, %s", s2)
				}
				if info.Heartbeat.Before(minTime) {
					t.Errorf("heartbeat last heartbeat time is before min time, %s", s2)
				}
				if info.CreatedAt.Before(minTime) {
					t.Errorf("heartbeat created time is before min time, %s", s2)
				}
			}
		}
	}
}
