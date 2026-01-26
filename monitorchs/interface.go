// Package monitorchs
//
// ----------------develop info----------------
//
//	@Author Calmu
//	@DateTime 2026-1-22 19:59
//
// --------------------------------------------
package monitorchs

import (
	"go.uber.org/zap"
	"sync"
)

// MonitorChsInterface 定义监控通道接口
type MonitorChsInterface interface {
	Run(wg *sync.WaitGroup)
	Stop()
	GetMonitorLog() []zap.Field
}
