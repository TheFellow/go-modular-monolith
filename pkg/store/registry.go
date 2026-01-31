package store

import (
	"reflect"
	"sync"
)

var (
	regMu sync.Mutex
	reg   = map[reflect.Type]any{}
)

// RegisterTypes registers bstore model types for the process. These are applied
// when opening a store.
func RegisterTypes(types ...any) {
	regMu.Lock()
	defer regMu.Unlock()

	for _, t := range types {
		if t == nil {
			continue
		}
		reg[reflect.TypeOf(t)] = t
	}
}

func registeredTypes() []any {
	regMu.Lock()
	defer regMu.Unlock()

	out := make([]any, 0, len(reg))
	for _, t := range reg {
		out = append(out, t)
	}
	return out
}
