package fyne_test

import (
	"errors"
	"sync"
	"testing"

	fyneui "github.com/TheFellow/go-modular-monolith/pkg/fyne"
	"github.com/TheFellow/go-modular-monolith/pkg/testutil/fynetest"
)

func TestLatestRequestRejectsOutOfOrderCompletion(t *testing.T) {
	executor := &fynetest.ManualExecutor{}
	request := fyneui.NewLatestRequest[string](executor, fyneui.InlineDispatcher{})
	var states []fyneui.LoadState[string]
	publish := func(state fyneui.LoadState[string]) { states = append(states, state) }

	request.Load(func() (string, error) { return "old", nil }, publish)
	request.Load(func() (string, error) { return "new", nil }, publish)
	if !executor.Run(1) || !executor.RunNext() {
		t.Fatal("expected two queued operations")
	}

	if len(states) != 3 || states[0].Status != fyneui.Loading || states[1].Status != fyneui.Loading ||
		states[2].Status != fyneui.Loaded || states[2].Value != "new" {
		t.Fatalf("unexpected publications: %#v", states)
	}
}

func TestLatestRequestChecksStalenessWhenUIPublicationRuns(t *testing.T) {
	executor := &fynetest.ManualExecutor{}
	dispatcher := &fynetest.ManualDispatcher{}
	request := fyneui.NewLatestRequest[int](executor, dispatcher)
	var values []int
	publish := func(state fyneui.LoadState[int]) {
		if state.Status == fyneui.Loaded {
			values = append(values, state.Value)
		}
	}

	request.Load(func() (int, error) { return 1, nil }, publish)
	executor.RunNext()
	request.Load(func() (int, error) { return 2, nil }, publish)
	executor.RunNext()
	dispatcher.Drain()

	if len(values) != 1 || values[0] != 2 {
		t.Fatalf("published values = %v, want [2]", values)
	}
}

func TestLatestRequestPublishesTypedFailure(t *testing.T) {
	want := errors.New("load failed")
	request := fyneui.NewLatestRequest[int](fyneui.InlineExecutor{}, fyneui.InlineDispatcher{})
	var got fyneui.LoadState[int]
	request.Load(func() (int, error) { return 0, want }, func(state fyneui.LoadState[int]) { got = state })
	if got.Status != fyneui.Failed || !errors.Is(got.Err, want) {
		t.Fatalf("state = %#v", got)
	}
}

func TestSubmissionRejectsDuplicateUntilPublication(t *testing.T) {
	executor := &fynetest.ManualExecutor{}
	dispatcher := &fynetest.ManualDispatcher{}
	submission := fyneui.NewSubmission(executor, dispatcher)
	runs := 0
	if !submission.Submit(func() error { runs++; return nil }, func(error) {}) {
		t.Fatal("first submission rejected")
	}
	if submission.Submit(func() error { runs++; return nil }, func(error) {}) {
		t.Fatal("duplicate submission accepted")
	}
	executor.RunNext()
	if !submission.Active() {
		t.Fatal("submission became inactive before UI publication")
	}
	dispatcher.Drain()
	if submission.Active() || runs != 1 {
		t.Fatalf("active=%v runs=%d", submission.Active(), runs)
	}
}

func TestSubmissionPublishesFailureAndBecomesReusable(t *testing.T) {
	want := errors.New("save failed")
	submission := fyneui.NewSubmission(fyneui.InlineExecutor{}, fyneui.InlineDispatcher{})
	var got error
	if !submission.Submit(func() error { return want }, func(err error) { got = err }) {
		t.Fatal("submission rejected")
	}
	if submission.Active() || !errors.Is(got, want) {
		t.Fatalf("active=%v error=%v", submission.Active(), got)
	}
	if !submission.Submit(func() error { return nil }, func(error) {}) {
		t.Fatal("submission was not reusable after failure")
	}
}

func TestSubmissionReleasesAfterPanickingWorkIsPublished(t *testing.T) {
	executor := &fynetest.ManualExecutor{}
	dispatcher := &fynetest.ManualDispatcher{}
	submission := fyneui.NewSubmission(executor, dispatcher)
	if !submission.Submit(func() error { panic("boom") }, func(error) {
		t.Fatal("panic must not be published as an ordinary error")
	}) {
		t.Fatal("submission rejected")
	}

	func() {
		defer func() {
			if recovered := recover(); recovered != "boom" {
				t.Fatalf("recovered %v, want boom", recovered)
			}
		}()
		executor.RunNext()
	}()
	if !submission.Active() {
		t.Fatal("submission became inactive before UI publication")
	}
	dispatcher.Drain()
	if submission.Active() {
		t.Fatal("submission remained active after panic publication")
	}
}

func TestLatestRequestIsRaceSafe(t *testing.T) {
	request := fyneui.NewLatestRequest[int](fyneui.InlineExecutor{}, fyneui.InlineDispatcher{})
	var publications sync.WaitGroup
	for i := 0; i < 100; i++ {
		publications.Add(1)
		go func(value int) {
			defer publications.Done()
			request.Load(func() (int, error) { return value, nil }, func(fyneui.LoadState[int]) {})
		}(i)
	}
	publications.Wait()
}
