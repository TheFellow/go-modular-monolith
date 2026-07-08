package middleware

// Handler processes an event.
//
// Dispatchers intentionally construct handler instances fresh for each event
// dispatch. Handler receiver fields are therefore suitable for event-local
// state, not shared service state.
type Handler[E any] interface {
	Handle(ctx *HandlerContext, event E) error
}

// PreparingHandler optionally queries data before Handle() runs.
//
// Dispatcher implementations should call Handling() for all handlers that
// implement it before calling Handle() on any handler for the same event.
// Handling() may capture event-local state on the handler receiver for that
// later Handle() call.
type PreparingHandler[E any] interface {
	Handler[E]
	Handling(ctx *HandlerContext, event E) error
}
