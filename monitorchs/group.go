// Package monitorchs
//
// ----------------develop info----------------
//
//	@Author Calmu
//	@DateTime 2026-1-23 16:14
//
// --------------------------------------------
package monitorchs

import (
	"github.com/calmu/hgotool/hlog"
	"go.uber.org/zap"
	"sync"
	"time"
)

// MonitorChsGroup 监控通道组
type MonitorChsGroup struct {
	groupChs        map[string]MonitorChsInterface
	quitCh          chan struct{}
	monitorDuration time.Duration
	hLog            hlog.HLoggerBase
}

type OptionsGroup func(*MonitorChsGroup)

func NewMonitorChsGroup(groupLen int, options ...OptionsGroup) *MonitorChsGroup {
	group := &MonitorChsGroup{
		groupChs: make(map[string]MonitorChsInterface, groupLen),
	}
	for _, option := range options {
		option(group)
	}

	// 确保在所有选项应用后仍有默认值
	if group.hLog == nil {
		group.hLog = hlog.GetLogger("default")
	}

	if group.monitorDuration == 0 {
		group.monitorDuration = MonitorDuration
	}

	return group
}

func WithGroupCh(name string, ch MonitorChsInterface) OptionsGroup {
	return func(group *MonitorChsGroup) {
		group.groupChs[name] = ch
	}
}

func WithGroupDuration(duration time.Duration) OptionsGroup {
	return func(group *MonitorChsGroup) {
		group.monitorDuration = duration
	}
}

func WithGroupLog(hLog hlog.HLoggerBase) OptionsGroup {
	return func(group *MonitorChsGroup) {
		group.hLog = hLog
	}
}

func WithGroupHLog() OptionsGroup {
	return func(group *MonitorChsGroup) {
		group.hLog = hlog.GlobalLoggers["default"]
	}
}

func (g *MonitorChsGroup) Start(wg *sync.WaitGroup) {
	g.quitCh = make(chan struct{}, 1)
	ticker := time.NewTicker(g.monitorDuration)
	go func() {
		defer wg.Done()
		for {
			select {
			case <-ticker.C:
				var zapFieldList []zap.Field
				for _, ch := range g.groupChs {
					zapFields := ch.GetMonitorLog()
					if zapFields != nil {
						zapFieldList = append(zapFieldList, zapFields...)
					}
				}
				if len(zapFieldList) > 0 && g.hLog != nil {
					g.hLog.Warn("chs monitor group", zapFieldList...)
				}
			case <-g.quitCh:
				ticker.Stop()
				return
			}
		}
	}()
}

func (g *MonitorChsGroup) Stop() {
	var once sync.Once
	once.Do(func() {
		if g.quitCh != nil {
			g.quitCh <- struct{}{}
			close(g.quitCh)
		}
	})
}
