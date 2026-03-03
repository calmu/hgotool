// Package counter
//
// ----------------develop info----------------
//
//	@Author Calmu
//	@DateTime 2026-2-13 17:27
//
// --------------------------------------------
package counter

import (
	"encoding/json"
	"github.com/calmu/hgotool/hticker"
	"github.com/calmu/hgotool/hwaitgroup"
	"math/rand"
	"testing"
	"time"
)

func TestSingle(t *testing.T) {
	c := NewCounter()

	c.AddSecond(1)
	c.AddSecond(2)

	var wgA hwaitgroup.WaitGroup

	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	addTicker := hticker.NewTicker(time.Second, hticker.WithTickFunc(func() {
		c.AddSecond(uint64(r.Intn(100))) // Intn returns a non-negative pseudo-random int n such that 0 <= n < 100.
	}))
	wgA.Go(addTicker.Start)

	secondTicker := hticker.NewTicker(time.Second, hticker.WithTickFunc(func() {
		sec := c.ResetSecond()
		t.Log("second:", sec)
		c.AddMinute(sec)
	}))
	wgA.Go(secondTicker.Start)

	minuteTicker := hticker.NewTicker(time.Minute, hticker.WithTickFunc(func() {
		minute := c.ResetMinute()
		t.Log("minute:", minute)
		c.AddHour(minute)
	}))
	wgA.Go(minuteTicker.Start)

	time.Sleep(time.Minute)

	addTicker.Stop()

	time.Sleep(time.Second)
	secondTicker.Stop()
	minuteTicker.Stop()
	wgA.Wait()

	js, _ := json.Marshal(c)
	t.Log("final", string(js))

	if c.LoadHour() <= 0 {
		t.Errorf("hour should be greater than 0")
	}
}

func TestSub(t *testing.T) {
	c := NewCounter()

	tickerList := make([]*hticker.Ticker, 0, 4)

	tickerList = append(tickerList, hticker.NewTicker(time.Second, hticker.WithQuitFunc(func() {
		t.Log("ticker stopped")
	}), hticker.WithTickFunc(func() {
		c.AddSecond(1)
	})))
	c.AddSecond(1)
	c.AddMinute(1)
	c.AddHour(1)

	c.Sub("test").AddMinute(1000)
	c.Sub("test").AddHour(1000)

	tickerList = append(tickerList, hticker.NewTicker(time.Second, hticker.WithTickFunc(func() {
		c.Sub("test").AddSecond(1)
	})))

	c.Sub("test2").AddMinute(1000)
	c.Sub("test2").AddHour(1000)

	tickerList = append(tickerList, hticker.NewTicker(time.Second, hticker.WithTickFunc(func() {
		c.Sub("test2").AddSecond(1)
	})))

	tickerList = append(tickerList, hticker.NewTicker(time.Second*5, hticker.WithTickFunc(func() {
		js, _ := json.Marshal(c)
		t.Log("counter:", string(js))
	})))

	var wg hwaitgroup.WaitGroup
	for _, ticker := range tickerList {
		wg.Go(ticker.Start)
	}
	time.Sleep(time.Second * 15)
	for _, ticker := range tickerList {
		ticker.Stop()
	}
	wg.Wait()
	js, _ := json.Marshal(c)
	t.Log("final", string(js))
}
