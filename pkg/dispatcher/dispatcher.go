//go:generate go run gen.go

package dispatcher

import "github.com/TheFellow/go-modular-monolith/pkg/middleware"

type Dispatcher struct{}

func New() *Dispatcher {
	return &Dispatcher{}
}

// handlerError is called when a handler returns an error.
// Return a non-nil error to stop dispatch immediately.
func (d *Dispatcher) handlerError(_ *middleware.Context, _ any, err error) error {
	return err
}

// unhandledEvent is called when an event is emitted but no handler exists for it.
func (d *Dispatcher) unhandledEvent(_ *middleware.Context, _ any) error {
	return nil
}
