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
		// 判断一下任务状态
		if task.state != StateStart {
			continue
		}
		if task.isRunning == false && task.runFunc != nil {
			if task.initFunc != nil {
				task.initFunc()
			}
			task.heartbeat = time.Now()
			wg.Add(1)
			go func() {
				defer func() {
					task.isRunning = false
					wg.Done()
				}()
				task.isRunning = true
				task.runFunc()
			}()
		}
	}
}

func (tg *TaskGroup) stop() {
	for _, task := range tg.tasks {
		if task.stopFunc != nil {
			task.stopTime = time.Now()
			task.stopFunc()
		}
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
		if tStr, err := hb.getHeartbeatFunc(key); err != nil && !errors.Is(err, redis.Nil) {
			// 应该记录日志
			continue
		} else {
			// 同时应该初始化状态到外部缓存
			if tStr == "" {
				task.state = StateStart
				hb.saveHeartbeatFunc(key, task.state)
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
