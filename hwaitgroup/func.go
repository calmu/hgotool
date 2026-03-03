// Package hwaitgroup
//
// ----------------develop info----------------
//
//	@Author xunmuhuang@rastar.com
//	@DateTime 2026-3-3 15:57
//
// --------------------------------------------
package hwaitgroup

import "sync"

// Go is a wrapper for sync.WaitGroup, providing a more convenient way to use sync.WaitGroup.Add and sync.WaitGroup.Done.
func Go(wg *sync.WaitGroup, f func()) {
	wg.Add(1)
	go func() {
		defer wg.Done()
		f()
	}()
}
