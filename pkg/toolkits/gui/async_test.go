//nolint:paralleltest // concurrency scheduling tests control their own goroutine ordering.
package gui_test

import (
	"errors"
	"sync"
	"testing"

	"github.com/TheFellow/go-modular-monolith/pkg/testutil"
	"github.com/TheFellow/go-modular-monolith/pkg/testutil/fynetest"
	gui "github.com/TheFellow/go-modular-monolith/pkg/toolkits/gui"
)

func TestLatestRequestRejectsOutOfOrderCompletion(t *testing.T) {
	executor := &fynetest.ManualExecutor{}
	request := gui.NewLatestRequest[string](executor, gui.InlineDispatcher{})
	var states []gui.LoadState[string]
	publish := func(state gui.LoadState[string]) { states = append(states, state) }

	request.Load(func() (string, error) { return "old", nil }, publish)
	request.Load(func() (string, error) { return "new", nil }, publish)
	if !executor.Run(1) || !executor.RunNext() {
		testutil.ErrorIf(t, true, "%v", "expected two queued operations")
	}

	if len(states) != 3 || states[0].Status != gui.Loading || states[1].Status != gui.Loading ||
		states[2].Status != gui.Loaded || states[2].Value != "new" {
		testutil.ErrorIf(t, true, "unexpected publications: %#v", states)
	}
}

func TestLatestRequestChecksStalenessWhenUIPublicationRuns(t *testing.T) {
	executor := &fynetest.ManualExecutor{}
	dispatcher := &fynetest.ManualDispatcher{}
	request := gui.NewLatestRequest[int](executor, dispatcher)
	var values []int
	publish := func(state gui.LoadState[int]) {
		if state.Status == gui.Loaded {
			values = append(values, state.Value)
		}
	}

	request.Load(func() (int, error) { return 1, nil }, publish)
	executor.RunNext()
	request.Load(func() (int, error) { return 2, nil }, publish)
	executor.RunNext()
	dispatcher.Drain()

	if len(values) != 1 || values[0] != 2 {
		testutil.ErrorIf(t, true, "published values = %v, want [2]", values)
	}
}

func TestLatestRequestPublishesTypedFailure(t *testing.T) {
	want := errors.New("load failed")
	request := gui.NewLatestRequest[int](gui.InlineExecutor{}, gui.InlineDispatcher{})
	var got gui.LoadState[int]
	request.Load(func() (int, error) { return 0, want }, func(state gui.LoadState[int]) { got = state })
	if got.Status != gui.Failed || !errors.Is(got.Err, want) {
		testutil.ErrorIf(t, true, "state = %#v", got)
	}
}

func TestSubmissionRejectsDuplicateUntilPublication(t *testing.T) {
	executor := &fynetest.ManualExecutor{}
	dispatcher := &fynetest.ManualDispatcher{}
	submission := gui.NewSubmission(executor, dispatcher)
	runs := 0
	if !submission.Submit(func() error { runs++; return nil }, func(error) {}) {
		testutil.ErrorIf(t, true, "%v", "first submission rejected")
	}
	if submission.Submit(func() error { runs++; return nil }, func(error) {}) {
		testutil.ErrorIf(t, true, "%v", "duplicate submission accepted")
	}
	executor.RunNext()
	if !submission.Active() {
		testutil.ErrorIf(t, true, "%v", "submission became inactive before UI publication")
	}
	dispatcher.Drain()
	if submission.Active() || runs != 1 {
		testutil.ErrorIf(t, true, "active=%v runs=%d", submission.Active(), runs)
	}
}

func TestSubmissionPublishesFailureAndBecomesReusable(t *testing.T) {
	want := errors.New("save failed")
	submission := gui.NewSubmission(gui.InlineExecutor{}, gui.InlineDispatcher{})
	var got error
	if !submission.Submit(func() error { return want }, func(err error) { got = err }) {
		testutil.ErrorIf(t, true, "%v", "submission rejected")
	}
	if submission.Active() || !errors.Is(got, want) {
		testutil.ErrorIf(t, true, "active=%v error=%v", submission.Active(), got)
	}
	if !submission.Submit(func() error { return nil }, func(error) {}) {
		testutil.ErrorIf(t, true, "%v", "submission was not reusable after failure")
	}
}

func TestSubmissionReleasesAfterPanickingWorkIsPublished(t *testing.T) {
	executor := &fynetest.ManualExecutor{}
	dispatcher := &fynetest.ManualDispatcher{}
	submission := gui.NewSubmission(executor, dispatcher)
	if !submission.Submit(func() error { panic("boom") }, func(error) {
		testutil.ErrorIf(t, true, "%v", "panic must not be published as an ordinary error")
	}) {
		testutil.ErrorIf(t, true, "%v", "submission rejected")
	}

	func() {
		defer func() {
			if recovered := recover(); recovered != "boom" {
				testutil.ErrorIf(t, true, "recovered %v, want boom", recovered)
			}
		}()
		executor.RunNext()
	}()
	if !submission.Active() {
		testutil.ErrorIf(t, true, "%v", "submission became inactive before UI publication")
	}
	dispatcher.Drain()
	if submission.Active() {
		testutil.ErrorIf(t, true, "%v", "submission remained active after panic publication")
	}
}

func TestLatestRequestIsRaceSafe(t *testing.T) {
	t.Parallel()

	request := gui.NewLatestRequest[int](gui.InlineExecutor{}, gui.InlineDispatcher{})
	var publications sync.WaitGroup
	for i := range 100 {
		publications.Add(1)
		go func(value int) {
			defer publications.Done()
			request.Load(func() (int, error) { return value, nil }, func(gui.LoadState[int]) {})
		}(i)
	}
	publications.Wait()
}
