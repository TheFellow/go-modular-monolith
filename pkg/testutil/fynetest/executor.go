// Package fynetest supplies deterministic collaborators and semantic widget
// driving for Fyne surface tests.
package fynetest

import "sync"

// ManualExecutor queues work until a test chooses its completion order.
type ManualExecutor struct {
	mu    sync.Mutex
	queue []func()
}

func (e *ManualExecutor) Execute(fn func()) {
	e.mu.Lock()
	e.queue = append(e.queue, fn)
	e.mu.Unlock()
}

func (e *ManualExecutor) Pending() int {
	e.mu.Lock()
	defer e.mu.Unlock()
	return len(e.queue)
}

// RunNext runs the oldest queued operation (FIFO).
func (e *ManualExecutor) RunNext() bool { return e.Run(0) }

// Run runs a queued operation by index, enabling out-of-order completion.
func (e *ManualExecutor) Run(index int) bool {
	e.mu.Lock()
	if index < 0 || index >= len(e.queue) {
		e.mu.Unlock()
		return false
	}
	fn := e.queue[index]
	e.queue = append(e.queue[:index], e.queue[index+1:]...)
	e.mu.Unlock()
	fn()
	return true
}

// ManualDispatcher queues UI publication for explicit test-controlled flushes.
type ManualDispatcher struct {
	mu    sync.Mutex
	queue []func()
}

func (d *ManualDispatcher) Dispatch(fn func()) {
	d.mu.Lock()
	d.queue = append(d.queue, fn)
	d.mu.Unlock()
}

func (d *ManualDispatcher) Pending() int {
	d.mu.Lock()
	defer d.mu.Unlock()
	return len(d.queue)
}

func (d *ManualDispatcher) RunNext() bool {
	d.mu.Lock()
	if len(d.queue) == 0 {
		d.mu.Unlock()
		return false
	}
	fn := d.queue[0]
	d.queue = d.queue[1:]
	d.mu.Unlock()
	fn()
	return true
}

func (d *ManualDispatcher) Drain() {
	for d.RunNext() {
	}
}
