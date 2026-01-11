package middleware

// Handler processes an event.
type Handler[E any] interface {
	Handle(ctx *Context, event E) error
}

// PreparingHandler optionally queries data before Handle() runs.
//
// Dispatcher implementations should call Handling() for all handlers that
// implement it before calling Handle() on any handler for the same event.
type PreparingHandler[E any] interface {
	Handler[E]
	Handling(ctx *Context, event E) error
}

