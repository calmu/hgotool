// Package hwaitgroup
//
// ----------------develop info----------------
//
//	@Author Calmu
//	@DateTime 2026-3-3 11:37
//
// --------------------------------------------
package hwaitgroup

import (
	"testing"
	"time"
)

func TestWgGo(t *testing.T) {
	var wg WaitGroup

	start := time.Now()
	wg.Go(func() {
		t.Log("wg go")
	})

	wg.Go(func() {
		time.Sleep(time.Second * 10)
		t.Log("wg go")
	})
	wg.Wait()

	now := time.Now()
	if !now.Add(-time.Second * 10).After(start) {
		t.Error("wg go error wait can not wait goroutine finish")
	}
}
