// Package hwaitgroup
//
// ----------------develop info----------------
//
//	@Author Calmu
//	@DateTime 2026-3-3 11:28
//
// --------------------------------------------
package hwaitgroup

import "sync"

type WaitGroup struct {
	sync.WaitGroup
}

// Go is a wrapper for sync.WaitGroup.Go, it's a more convenient way to use sync.WaitGroup.Add and sync.WaitGroup.Done. if go version >= 1.25, you can use sync.WaitGroup.Go directly
func (wg *WaitGroup) Go(f func()) {
	wg.Add(1)
	go func() {
		defer wg.Done()
		f()
	}()
}
