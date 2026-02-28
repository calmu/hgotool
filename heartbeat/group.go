// Package heartbeat
//
// ----------------develop info----------------
//
//	@Author Calmu
//	@DateTime 2025-2-24 15:15
//
// --------------------------------------------
package heartbeat

import (
	"errors"
	"fmt"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"sync"
	"time"
)

type TaskGroup struct {
	group string         // task group name
	tasks []*Task        // task list
	names map[string]int // task name index map
	state string         // task group state, default is StateStart, can be StateStart, StatePause, StateStop
}

type GroupOption func(*TaskGroup)

func WithGroupName(name string) GroupOption {
	return func(tg *TaskGroup) {
		tg.group = name
	}
}

func WithTaskList(tasks ...*Task) GroupOption {
	return func(tg *TaskGroup) {
		tg.tasks = tasks
		tg.names = make(map[string]int, len(tasks))
		for i, t := range tasks {
			tg.names[t.name] = i
		}
	}
}

func NewTaskGroup(options ...GroupOption) *TaskGroup {
	tg := &TaskGroup{}
	for _, option := range options {
		option(tg)
	}
	return tg
}

func (tg *TaskGroup) run(wg *sync.WaitGroup) {
	// 判断一下组状态
	if tg.state != StateStart {
		return
	}
	for _, task := range tg.tasks {
		tg.runTask(wg, task)
	}
}

func (tg *TaskGroup) runTask(wg *sync.WaitGroup, task *Task) {
	task.lock.Lock()
	defer task.lock.Unlock()
	// 判断一下任务状态
	if task.state != StateStart {
		return
	}
	if task.isRunning == false && task.runFunc != nil {
		if task.initFunc != nil {
			task.initFunc()
		}
		task.heartbeat = time.Now()
		task.isRunning = true
		wg.Add(1)
		go func() {
			defer func() {
				task.lock.Lock()
				defer task.lock.Unlock()

				task.isRunning = false
				wg.Done()
			}()
			// 如果心跳任务需要定时运行，则启动任务
			if task.runTickerFlag {
				defer task.StopHeartbeat()

				task.StartHeartbeat()
			}
			task.runFunc()
		}()
	}
}

func (tg *TaskGroup) stop(state string) {
	for _, task := range tg.tasks {
		task.stop(state)
	}
}

func (tg *TaskGroup) buildStateKey() string {
	return fmt.Sprintf("%s:state", tg.group)
}

func (tg *TaskGroup) buildTaskStateKey(task *Task) string {
	return fmt.Sprintf("%s:%s:state", tg.group, task.name)
}

func (tg *TaskGroup) syncTasksStateFromHeartbeat(hb *Heartbeat) {
	for _, task := range tg.tasks {
		key := tg.buildTaskStateKey(task)
		if tStr, err := hb.getHeartbeatFunc(hb.buildCacheKey(key)); err != nil && !errors.Is(err, redis.Nil) { // Fixed: Added missing argument
			if hb.logger != nil {
				hb.logger.Error("get task state error", zap.Error(err), zap.String("key", key))
			}
			continue
		} else {
			// 同时应该初始化状态到外部缓存
			if tStr == "" {
				task.state = StateStart
				hb.saveHeartbeatFunc(hb.buildCacheKey(key), task.state) // Fixed: Added missing argument
			}
			// 操作任务
			switch tStr {
			case StateStart, StatePause, StateStop:
				task.state = tStr
			}
		}
	}
}

func (tg *TaskGroup) buildTaskInfoKey(task *Task) string {
	return fmt.Sprintf("%s:%s:info", tg.group, task.name)
}
