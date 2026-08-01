//nolint:paralleltest // concurrency scheduling tests control their own goroutine ordering.
package gui_test

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/TheFellow/go-modular-monolith/pkg/testutil"
	gui "github.com/TheFellow/go-modular-monolith/pkg/toolkits/gui"
)

func TestAsyncExecutorCompletesBackgroundWork(t *testing.T) {
	done := make(chan struct{})
	gui.AsyncExecutor{}.Execute(func() { close(done) })
	select {
	case <-done:
	case <-time.After(time.Second):
		testutil.Fail(t, "%v", "asynchronous work did not complete")
	}
}

func TestManagedExecutorCloseDrainsAcceptedWorkAndRejectsLaterWork(t *testing.T) {
	executor := gui.NewManagedExecutor()
	started := make(chan struct{})
	release := make(chan struct{})
	finished := make(chan struct{})
	testutil.ErrorIf(t, !executor.TryExecute(func() {
		close(started)
		<-release
		close(finished)
	}), "%v", "open executor rejected work")
	<-started

	closed := make(chan struct{})
	go func() {
		executor.Close()
		close(closed)
	}()
	for executor.TryExecute(func() {}) {
		// An operation racing Close may be accepted. Rejection proves Close has
		// closed admission before we assert the stable post-close behavior.
	}
	testutil.ErrorIf(t, executor.TryExecute(func() { t.Error("post-close work ran") }), "%v", "executor accepted work after its admission gate closed")
	select {
	case <-closed:
		testutil.Fail(t, "%v", "close returned before accepted work completed")
	default:
	}
	close(release)
	<-closed
	<-finished
}

func TestManagedExecutorCloseCancelsContextWork(t *testing.T) {
	executor := gui.NewManagedExecutor()
	started, cancelled := make(chan struct{}), make(chan struct{})
	testutil.ErrorIf(t, !executor.ExecuteContext(func(ctx context.Context) {
		close(started)
		<-ctx.Done()
		close(cancelled)
	}), "%v", "open executor rejected context work")
	<-started
	executor.Close()
	<-cancelled
}

func TestManagedExecutorConcurrentExecuteAndCloseDrainsEveryAcceptedOperation(t *testing.T) {
	executor := gui.NewManagedExecutor()
	const callers = 64
	start := make(chan struct{})
	var callersDone sync.WaitGroup
	var accepted atomic.Int64
	var completed atomic.Int64
	callersDone.Add(callers)
	for range callers {
		go func() {
			defer callersDone.Done()
			<-start
			if executor.TryExecute(func() { completed.Add(1) }) {
				accepted.Add(1)
			}
		}()
	}
	close(start)
	executor.Close()
	callersDone.Wait()
	executor.Close()
	{
		got, want := completed.Load(), accepted.Load()
		testutil.ErrorIf(t, got != want, "completed %d operations, want all %d accepted operations", got, want)
	}
}
