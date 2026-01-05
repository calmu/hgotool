// Package monitorchs
//
// ----------------develop info----------------
//
//	@Author xunmuhuang@rastar.com
//	@DateTime 2026-1-5 17:37
//
// --------------------------------------------
package monitorchs

import (
	"github.com/calmu/hgotool/hlog"
	"sync"
	"testing"
	"time"
)

func TestMonitorChs(t *testing.T) {
	chs := make([]chan string, 0, 10)
	for i := 0; i < 10; i++ {
		chs = append(chs, make(chan string, 100))
	}

	// 向通道发送一些数据以测试监控
	for i := 0; i < 5; i++ {
		select {
		case chs[i] <- "test data":
		default:
			t.Logf("Channel %d is full", i)
		}
	}

	// 初始化hlog
	hlog.InitRotatingLogger("default", hlog.RotateConfig{
		Level:        "info",
		Encoder:      "json",
		OutputType:   "file",
		Filename:     "./log/rotated/app.log",
		TimeRotation: "daily", // 按天轮转
		MaxSize:      1,       // 1MB后轮转
		MaxBackups:   3,       // 保留3个备份
		MaxAge:       7,       // 保留7天
	})

	m := NewMonitorChs(WithChs("test", chs), WithDuration[string](time.Second*5))

	var wg sync.WaitGroup
	wg.Add(1)
	m.Run(&wg)

	// 等待一段时间以观察监控效果
	time.Sleep(time.Second * 6)

	m.Stop()
	wg.Wait()
}

func TestMonitorChsInt(t *testing.T) {
	chs := make([]chan int, 0, 5)
	for i := 0; i < 5; i++ {
		chs = append(chs, make(chan int, 50))
	}

	// 向通道发送一些数据
	for i := 0; i < 3; i++ {
		select {
		case chs[i] <- i:
		default:
			t.Logf("Channel %d is full", i)
		}
	}

	// 初始化hlog
	hlog.InitRotatingLogger("default", hlog.RotateConfig{
		Level:        "info",
		Encoder:      "json",
		OutputType:   "file",
		Filename:     "./log/rotated/app.log",
		TimeRotation: "daily", // 按天轮转
		MaxSize:      1,       // 1MB后轮转
		MaxBackups:   3,       // 保留3个备份
		MaxAge:       7,       // 保留7天
	})

	m := NewMonitorChs(WithChs("intTest", chs), WithDuration[int](time.Second*3))

	var wg sync.WaitGroup
	wg.Add(1)
	m.Run(&wg)

	// 等待一段时间以观察监控效果
	time.Sleep(time.Second * 4)

	m.Stop()
	wg.Wait()
}

func TestMonitorChsMultipleTypes(t *testing.T) {
	// 测试不同类型通道的监控
	stringChs := []chan string{make(chan string, 10), make(chan string, 10)}
	intChs := []chan int{make(chan int, 10), make(chan int, 10)}

	// 发送一些数据
	select {
	case stringChs[0] <- "test":
	default:
		t.Log("String channel is full")
	}
	select {
	case intChs[0] <- 123:
	default:
		t.Log("Int channel is full")
	}

	// 初始化hlog
	hlog.InitRotatingLogger("default", hlog.RotateConfig{
		Level:        "info",
		Encoder:      "json",
		OutputType:   "file",
		Filename:     "./log/rotated/app.log",
		TimeRotation: "daily", // 按天轮转
		MaxSize:      1,       // 1MB后轮转
		MaxBackups:   3,       // 保留3个备份
		MaxAge:       7,       // 保留7天
	})

	// 创建两个监控器，分别监控不同类型的通道
	stringMonitor := NewMonitorChs(WithChs("string", stringChs), WithDuration[string](time.Second*5))
	intMonitor := NewMonitorChs(WithChs("int", intChs), WithDuration[int](time.Second*5))

	var wg sync.WaitGroup
	wg.Add(2)

	stringMonitor.Run(&wg)
	intMonitor.Run(&wg)

	time.Sleep(time.Second * 6)

	stringMonitor.Stop()
	intMonitor.Stop()

	wg.Wait()
}
