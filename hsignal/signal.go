// Package hsignal
//
// ----------------develop info----------------
//
//	@Author Calmu
//	@DateTime 2025-2-25 16:59
//
// --------------------------------------------
package hsignal

import (
	"context"
	"go.uber.org/zap"
	"os"
	"os/signal"
	"syscall"
)

// ContextSignal 创建一个信号处理
func ContextSignal(logger func(string, ...zap.Field)) (context.Context, context.CancelFunc) {
	ctx, cancel := context.WithCancel(context.Background())
	signalCh := make(chan os.Signal, 1)
	signal.Notify(signalCh, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	go func() {
		for {
			//监听退出信号
			select {
			case sig, _ := <-signalCh: //接收到退出信号
				if logger != nil {
					logger("signal received", zap.Any("signal", sig))
				}
				cancel()
				return
			}
		}
	}()
	return ctx, cancel
}
