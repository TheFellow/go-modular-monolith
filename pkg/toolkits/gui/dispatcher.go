package gui

import "sync"

import framework "fyne.io/fyne/v2"

// Dispatcher schedules state publication on Fyne's UI goroutine. Domain view
// models accept this small interface so asynchronous behavior remains
// deterministic under test.
type Dispatcher interface {
	Dispatch(func())
}

// MainDispatcher publishes work through Fyne's event loop.
type MainDispatcher struct{}

func (MainDispatcher) Dispatch(fn func()) { framework.Do(fn) }

// InlineDispatcher runs work synchronously. It is intended for tests and other
// environments that deliberately own serialization.
type InlineDispatcher struct{}

func (InlineDispatcher) Dispatch(fn func()) { fn() }

// GatedDispatcher owns publication lifetime independently from background
// work. A callback checks the gate both before it is queued and immediately
// before it runs, ensuring already-queued UI work is dropped after Close.
type GatedDispatcher struct {
	dispatcher Dispatcher
	mu         sync.Mutex
	closed     bool
	active     int
	drained    *sync.Cond
}

// NewGatedDispatcher wraps a dispatcher with a closable publication gate.
func NewGatedDispatcher(dispatcher Dispatcher) *GatedDispatcher {
	d := &GatedDispatcher{dispatcher: dispatcher}
	d.drained = sync.NewCond(&d.mu)
	return d
}

func (d *GatedDispatcher) Dispatch(fn func()) {
	d.mu.Lock()
	closed := d.closed
	d.mu.Unlock()
	if closed {
		return
	}
	d.dispatcher.Dispatch(func() {
		d.mu.Lock()
		if d.closed {
			d.mu.Unlock()
			return
		}
		d.active++
		d.mu.Unlock()
		defer func() {
			d.mu.Lock()
			d.active--
			if d.active == 0 {
				d.drained.Broadcast()
			}
			d.mu.Unlock()
		}()
		fn()
	})
}

// Close prevents future and already-queued callbacks from publishing. It
// waits for a callback currently inside the gate to return.
func (d *GatedDispatcher) Close() {
	d.mu.Lock()
	d.closed = true
	for d.active != 0 {
		d.drained.Wait()
	}
	d.mu.Unlock()
}
