package middleware

import (
	"log"
	"sync"

	cedar "github.com/cedar-policy/cedar-go"
)

var (
	dispatcherOnce sync.Once
	dispatcher     EventDispatcher
)

// SetEventDispatcher configures the process-wide event dispatcher used by the
// default command chain. It is safe to call multiple times; only the first call
// wins.
func SetEventDispatcher(d EventDispatcher) {
	dispatcherOnce.Do(func() {
		dispatcher = d
	})
}

// DispatchEvents dispatches any events collected on the middleware context
// after the command completes.
func DispatchEvents() CommandMiddleware {
	return func(ctx *Context, _ cedar.EntityUID, _ cedar.Entity, next CommandNext) error {
		if err := next(ctx); err != nil {
			return err
		}

		if dispatcher == nil {
			return nil
		}

		for _, event := range ctx.Events() {
			if err := dispatcher.Dispatch(ctx, event); err != nil {
				log.Printf("handler error for %T: %v", event, err)
			}
		}
		return nil
	}
}
