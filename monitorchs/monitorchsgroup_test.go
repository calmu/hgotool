// Package monitorchs
//
// ----------------develop info----------------
//
//	@Author Calmu
//	@DateTime 2026-1-26 10:37
//
// --------------------------------------------
package monitorchs

import (
	"github.com/calmu/hgotool/hlog"
	"github.com/calmu/hgotool/hwaitgroup"
	"testing"
	"time"
)

func TestMonitorChsGroup(t *testing.T) {
	strChs := make([]chan string, 0, 5)
	for i := 0; i < 5; i++ {
		strChs = append(strChs, make(chan string, 100))
	}

	for i, ch := range strChs {
		select {
		case ch <- "hello":
		default:
			t.Logf("channel %d is full", i)
		}
	}

	intChs := make([]chan int, 0, 5)
	for i := 0; i < 5; i++ {
		intChs = append(intChs, make(chan int, 100))
	}

	for i, ch := range intChs {
		select {
		case ch <- i:
		default:
			t.Logf("channel %d is full", i)
		}
	}

	// 初始化hlog
	hlog.InitRotatingLogger("default", hlog.RotateConfig{
		Level:        "info",
		Encoder:      "json",
		OutputType:   "both",
		Filename:     "./log/rotated/app.log",
		TimeRotation: "daily", // 按天轮转
		MaxSize:      1,       // 1MB后轮转
		MaxBackups:   3,       // 保留3个备份
		MaxAge:       7,       // 保留7天
		EncoderConfig: &hlog.EncoderConfig{
			EncodeTime: "iso8601",
		},
	})
	defer hlog.Close()

	monitorChsGroup := NewMonitorChsGroup(2,
		WithGroupCh("str", NewMonitorChs[string](WithChs("str", strChs))),
		WithGroupCh("int", NewMonitorChs[int](WithChs("int", intChs))),
		WithGroupDuration(time.Second*5),
	)
	var wg hwaitgroup.WaitGroup
	wg.Go(monitorChsGroup.Start)

	time.Sleep(time.Second * 15)
	monitorChsGroup.Stop()
	wg.Wait()
}

func TestMonitorChsGroupWithStruct(t *testing.T) {
	type TestStruct struct {
		Name string
		Age  int
	}
	tChs := make([]chan TestStruct, 0, 5)
	for i := 0; i < 5; i++ {
		tChs = append(tChs, make(chan TestStruct, 100))
	}

	type TestStruct2 struct {
		Job   string
		Years int
	}

	t2Chs := make([]chan TestStruct2, 0, 5)
	for i := 0; i < 5; i++ {
		t2Chs = append(t2Chs, make(chan TestStruct2, 100))
	}

	strChs := make([]chan string, 0, 5)
	for i := 0; i < 5; i++ {
		strChs = append(strChs, make(chan string, 100))
	}

	// 初始化hlog
	hlog.InitRotatingLogger("default", hlog.RotateConfig{
		Level:        "info",
		Encoder:      "json",
		OutputType:   "both",
		Filename:     "./log/rotated/app.log",
		TimeRotation: "daily", // 按天轮转
		MaxSize:      1,       // 1MB后轮转
		MaxBackups:   3,       // 保留3个备份
		MaxAge:       7,       // 保留7天
		EncoderConfig: &hlog.EncoderConfig{
			EncodeTime: "iso8601",
		},
	})
	defer hlog.Close()

	monitorChsGroup := NewMonitorChsGroup(2,
		WithGroupCh("test", NewMonitorChs[TestStruct](WithChs("t1", tChs))),
		WithGroupCh("test2", NewMonitorChs[TestStruct2](WithChs("ts", t2Chs))),
		WithGroupCh("str", NewMonitorChs[string](WithChs("str", strChs))),
		WithGroupDuration(time.Second*5),
	)

	var wg hwaitgroup.WaitGroup
	wg.Go(monitorChsGroup.Start)

	for i := 0; i < 10; i++ {
		select {
		case tChs[i%5] <- TestStruct{Name: "test", Age: i}:
		default:
			t.Logf("channel tChs%d is full", i%5)
		}
		time.Sleep(time.Second)
		select {
		case t2Chs[i%5] <- TestStruct2{Job: "test", Years: i}:
		default:
			t.Logf("channel tChs%d is full", i%5)
		}
		select {
		case strChs[i%5] <- "hello":
		default:
			t.Logf("channel tChs%d is full", i%5)
		}
	}

	time.Sleep(time.Second * 5)
	monitorChsGroup.Stop()
	wg.Wait()
}
