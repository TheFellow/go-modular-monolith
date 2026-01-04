package store

import (
	"context"
	"reflect"
	"sync"
)

var (
	regMu sync.Mutex
	reg   = map[reflect.Type]any{}
)

// RegisterTypes registers bstore model types for the process.
//
// Intended usage is from init() functions in internal DAO packages so callers do
// not need to assemble a type list.
func RegisterTypes(types ...any) {
	var newlyAdded []any

	regMu.Lock()
	for _, t := range types {
		if t == nil {
			continue
		}
		rt := reflect.TypeOf(t)
		if _, ok := reg[rt]; ok {
			continue
		}
		reg[rt] = t
		newlyAdded = append(newlyAdded, t)
	}
	regMu.Unlock()

	if len(newlyAdded) == 0 {
		return
	}

	mu.Lock()
	db := DB
	mu.Unlock()

	if db == nil {
		return
	}
	if err := db.Register(context.Background(), newlyAdded...); err != nil {
		panic(err)
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
