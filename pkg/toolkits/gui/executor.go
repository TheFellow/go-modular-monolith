package gui

import (
	"context"
	"sync"
)

// Executor starts background application work. Production surfaces normally
// use AsyncExecutor; tests inject a deterministic executor.
type Executor interface {
	Execute(func())
}

// AsyncExecutor starts each operation in its own goroutine.
type AsyncExecutor struct{}

func (AsyncExecutor) Execute(fn func()) { go fn() }

// InlineExecutor executes work before returning. It is useful for small tests
// that do not need to control completion order.
type InlineExecutor struct{}

func (InlineExecutor) Execute(fn func()) { fn() }

// ManagedExecutor owns the lifetime of background presentation work. Close
// atomically stops admission and waits for every operation accepted before the
// gate closed. Execute retains the small surface-facing Executor contract;
// TryExecute is available to owners that need to observe rejection.
type ManagedExecutor struct {
	mu     sync.Mutex
	closed bool
	work   sync.WaitGroup
	ctx    context.Context
	cancel context.CancelFunc
}

// NewManagedExecutor constructs an executor that owns its goroutines.
func NewManagedExecutor() *ManagedExecutor {
	ctx, cancel := context.WithCancel(context.Background())
	return &ManagedExecutor{ctx: ctx, cancel: cancel}
}

// Execute starts fn when the executor is open and silently rejects it after
// Close. Surface code deliberately does not need to coordinate application
// shutdown.
func (e *ManagedExecutor) Execute(fn func()) { e.TryExecute(fn) }

// ExecuteContext starts work with a context cancelled when the executor
// closes. LatestRequest uses this optional extension so shutdown can interrupt
// application work instead of merely waiting for it.
func (e *ManagedExecutor) ExecuteContext(fn func(context.Context)) bool {
	return e.TryExecute(func() { fn(e.ctx) })
}

// TryExecute reports whether fn was accepted for background execution.
func (e *ManagedExecutor) TryExecute(fn func()) bool {
	e.mu.Lock()
	if e.closed {
		e.mu.Unlock()
		return false
	}
	e.work.Add(1)
	e.mu.Unlock()

	go func() {
		defer e.work.Done()
		fn()
	}()
	return true
}

// Wait waits for all work accepted so far. Application shutdown should
// normally use Close so no later work can race with the wait.
func (e *ManagedExecutor) Wait() { e.work.Wait() }

// Close stops admission and drains all accepted work. It is idempotent and
// safe to call concurrently with Execute and other Close calls.
func (e *ManagedExecutor) Close() {
	e.mu.Lock()
	e.closed = true
	e.cancel()
	e.mu.Unlock()
	e.work.Wait()
}
