// Package hticker
//
// ----------------develop info----------------
//
//	@Author Calmu
//	@DateTime 2025-2-14 15:20
//
// --------------------------------------------
package hticker

import (
	"fmt"
	"time"
)

type Ticker struct {
	Ticker      *time.Ticker
	Quit        chan struct{}
	deferFunc   func()
	quitFunc    func()
	tickFunc    func()
	isGoroutine bool
}

type Option func(*Ticker)

func WithGoroutine(goroutine bool) Option {
	return func(t *Ticker) {
		t.isGoroutine = goroutine
	}
}

func WithDeferFunc(f func()) Option {
	return func(t *Ticker) {
		t.deferFunc = f
	}
}

func WithTickFunc(f func()) Option {
	return func(t *Ticker) {
		t.tickFunc = f
	}
}

func WithQuitFunc(f func()) Option {
	return func(t *Ticker) {
		t.quitFunc = f
	}
}

func (t *Ticker) Start() {
	if t.isGoroutine {
		go t.start()
	} else {
		t.start()
	}
}

func (t *Ticker) start() {
	defer func() {
		if t.deferFunc != nil {
			t.deferFunc()
		}
	}()
	for {
		select {
		case <-t.Quit:
			if t.quitFunc != nil {
				t.quitFunc()
			}
			return
		case <-t.Ticker.C:
			t.tickFunc()
		}
	}
}

func NewTicker(d time.Duration, options ...Option) *Ticker {
	t := time.NewTicker(d)
	ticker := &Ticker{Ticker: t, Quit: make(chan struct{}, 1), isGoroutine: true, tickFunc: func() {
		fmt.Println("tick")
	}}
	for _, option := range options {
		option(ticker)
	}

	return ticker
}

func (t *Ticker) Stop() {
	t.Ticker.Stop()
	close(t.Quit)
}
