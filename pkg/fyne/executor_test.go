//nolint:paralleltest // concurrency scheduling tests control their own goroutine ordering.
package fyne_test

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"

	fyneui "github.com/TheFellow/go-modular-monolith/pkg/fyne"
)

func TestAsyncExecutorCompletesBackgroundWork(t *testing.T) {
	done := make(chan struct{})
	fyneui.AsyncExecutor{}.Execute(func() { close(done) })
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("asynchronous work did not complete")
	}
}

func TestManagedExecutorCloseDrainsAcceptedWorkAndRejectsLaterWork(t *testing.T) {
	executor := fyneui.NewManagedExecutor()
	started := make(chan struct{})
	release := make(chan struct{})
	finished := make(chan struct{})
	if !executor.TryExecute(func() {
		close(started)
		<-release
		close(finished)
	}) {
		t.Fatal("open executor rejected work")
	}
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
	if executor.TryExecute(func() { t.Error("post-close work ran") }) {
		t.Fatal("executor accepted work after its admission gate closed")
	}
	select {
	case <-closed:
		t.Fatal("close returned before accepted work completed")
	default:
	}
	close(release)
	<-closed
	<-finished
}

func TestManagedExecutorConcurrentExecuteAndCloseDrainsEveryAcceptedOperation(t *testing.T) {
	executor := fyneui.NewManagedExecutor()
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
	if got, want := completed.Load(), accepted.Load(); got != want {
		t.Fatalf("completed %d operations, want all %d accepted operations", got, want)
	}
}
