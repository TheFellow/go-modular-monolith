//go:generate go run gen.go

package dispatcher

import (
	"reflect"
	"strings"

	"github.com/TheFellow/go-modular-monolith/pkg/log"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
)

type Dispatcher struct{}

func New() *Dispatcher {
	return &Dispatcher{}
}

// handlerError is called when a handler returns an error.
// Return a non-nil error to stop dispatch immediately.
func (d *Dispatcher) handlerError(ctx *middleware.Context, event any, err error) error {
	return err
}

// unhandledEvent is called when an event is emitted but no handler exists for it.
func (d *Dispatcher) unhandledEvent(ctx *middleware.Context, event any) error {
	if ctx != nil {
		log.FromContext(ctx).Warn(
			"unhandled event",
			log.EventType(eventTypeLabel(event)),
		)
	}
	return nil
}

func eventTypeLabel(event any) string {
	t := reflect.TypeOf(event)
	if t == nil {
		return ""
	}
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	pkg := t.PkgPath()
	if pkg != "" {
		if i := strings.LastIndex(pkg, "/"); i >= 0 && i < len(pkg)-1 {
			pkg = pkg[i+1:]
		}
	}
	if pkg == "" {
		return t.Name()
	}
	return pkg + "." + t.Name()
}
