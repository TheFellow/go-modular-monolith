//nolint:paralleltest // Fyne's application driver is process-global.
package gui

import (
	"testing"
	"time"

	"fyne.io/fyne/v2/test"
	"github.com/TheFellow/go-modular-monolith/pkg/testutil"
)

func TestInlineDispatcherPublishesSynchronously(t *testing.T) {
	called := false
	InlineDispatcher{}.Dispatch(func() { called = true })
	if !called {
		testutil.ErrorIf(t, true, "%v", "inline dispatcher did not publish before returning")
	}
}

func TestMainDispatcherPublishesThroughFyneDriver(t *testing.T) {
	app := test.NewApp()
	t.Cleanup(app.Quit)
	called := make(chan struct{})
	MainDispatcher{}.Dispatch(func() { close(called) })
	select {
	case <-called:
	case <-time.After(time.Second):
		testutil.ErrorIf(t, true, "%v", "main dispatcher did not publish through the Fyne driver")
	}
}

type queuedDispatcher struct{ callbacks []func() }

func (d *queuedDispatcher) Dispatch(fn func()) { d.callbacks = append(d.callbacks, fn) }

func TestGatedDispatcherDropsQueuedAndFuturePublicationsAfterClose(t *testing.T) {
	queue := &queuedDispatcher{}
	dispatcher := NewGatedDispatcher(queue)
	var published int
	dispatcher.Dispatch(func() { published++ })
	if len(queue.callbacks) != 1 {
		testutil.ErrorIf(t, true, "queued %d callbacks, want 1", len(queue.callbacks))
	}
	dispatcher.Close()
	queue.callbacks[0]()
	dispatcher.Dispatch(func() { published++ })
	if published != 0 {
		testutil.ErrorIf(t, true, "published %d callbacks after close, want 0", published)
	}
	if len(queue.callbacks) != 1 {
		testutil.ErrorIf(t, true, "queued %d callbacks after close, want 1", len(queue.callbacks))
	}
}

func TestGatedDispatcherAllowsReentrantDispatchWhileCloseWaits(t *testing.T) {
	t.Parallel()

	dispatcher := NewGatedDispatcher(InlineDispatcher{})
	entered := make(chan struct{})
	release := make(chan struct{})
	done := make(chan struct{})

	go dispatcher.Dispatch(func() {
		close(entered)
		<-release
		dispatcher.Dispatch(func() {})
		close(done)
	})
	<-entered
	closed := make(chan struct{})
	go func() {
		dispatcher.Close()
		close(closed)
	}()
	close(release)
	<-done
	<-closed
}
