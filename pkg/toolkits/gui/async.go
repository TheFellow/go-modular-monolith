package gui

import "sync"

// LoadStatus describes the lifecycle of an asynchronous value.
type LoadStatus uint8

const (
	Idle LoadStatus = iota
	Loading
	Loaded
	Failed
)

// LoadState is a framework-independent snapshot published by LatestRequest.
// Err is populated only for Failed; Value is populated only for Loaded.
type LoadState[T any] struct {
	Status LoadStatus
	Value  T
	Err    error
}

// LatestRequest executes loads and publishes only the newest completion. It is
// safe to call from multiple goroutines. Staleness is checked on the UI
// dispatcher, not merely when background work completes, because publication
// itself may be queued.
type LatestRequest[T any] struct {
	executor   Executor
	dispatcher Dispatcher

	mu         sync.Mutex
	generation uint64
}

func NewLatestRequest[T any](executor Executor, dispatcher Dispatcher) *LatestRequest[T] {
	return &LatestRequest[T]{executor: executor, dispatcher: dispatcher}
}

// Load starts work, immediately publishes Loading, and returns its generation.
func (r *LatestRequest[T]) Load(work func() (T, error), publish func(LoadState[T])) uint64 {
	r.mu.Lock()
	r.generation++
	generation := r.generation
	r.mu.Unlock()

	r.dispatch(generation, func() { publish(LoadState[T]{Status: Loading}) })
	r.executor.Execute(func() {
		value, err := work()
		r.dispatch(generation, func() {
			if err != nil {
				publish(LoadState[T]{Status: Failed, Err: err})
				return
			}
			publish(LoadState[T]{Status: Loaded, Value: value})
		})
	})
	return generation
}

// Invalidate prevents all currently queued publications from being observed.
func (r *LatestRequest[T]) Invalidate() {
	r.mu.Lock()
	r.generation++
	r.mu.Unlock()
}

func (r *LatestRequest[T]) dispatch(generation uint64, fn func()) {
	r.dispatcher.Dispatch(func() {
		r.mu.Lock()
		current := generation == r.generation
		r.mu.Unlock()
		if current {
			fn()
		}
	})
}

// Submission runs at most one mutation at a time. Submit returns false without
// scheduling work when another submission is active.
type Submission struct {
	executor   Executor
	dispatcher Dispatcher
	mu         sync.Mutex
	active     bool
}

func NewSubmission(executor Executor, dispatcher Dispatcher) *Submission {
	return &Submission{executor: executor, dispatcher: dispatcher}
}

func (s *Submission) Submit(work func() error, publish func(error)) bool {
	s.mu.Lock()
	if s.active {
		s.mu.Unlock()
		return false
	}
	s.active = true
	s.mu.Unlock()

	s.executor.Execute(func() {
		defer func() {
			if recovered := recover(); recovered != nil {
				// Preserve the panic for the executor's owner while still making
				// the submission usable again once the UI processes completion.
				s.dispatcher.Dispatch(s.release)
				panic(recovered)
			}
		}()
		err := work()
		s.dispatcher.Dispatch(func() {
			s.release()
			publish(err)
		})
	})
	return true
}

func (s *Submission) release() {
	s.mu.Lock()
	s.active = false
	s.mu.Unlock()
}

func (s *Submission) Active() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.active
}
