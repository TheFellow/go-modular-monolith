package app

import (
	"github.com/TheFellow/go-modular-monolith/pkg/optional"
	"github.com/TheFellow/go-modular-monolith/pkg/store"
)

type Option func(*App)

func WithStore(s *store.Store) Option {
	return func(a *App) {
		if a == nil {
			return
		}
		if s == nil {
			a.Store = optional.None[*store.Store]()
			return
		}
		a.Store = optional.Some(s)
	}
}
