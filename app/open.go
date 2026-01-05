package app

import (
	"context"
	"sync"

	"github.com/TheFellow/go-modular-monolith/pkg/dispatcher"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
	"github.com/TheFellow/go-modular-monolith/pkg/store"
	cedar "github.com/cedar-policy/cedar-go"
)

var configureOnce sync.Once

func Open(path string) (*App, error) {
	configureOnce.Do(func() {
		middleware.SetEventDispatcher(dispatcher.New())
	})

	s, err := store.Open(path)
	if err != nil {
		return nil, err
	}

	a := New()
	a.Store = s
	return a, nil
}

func (a *App) Close() error {
	if a == nil || a.Store == nil {
		return nil
	}
	return a.Store.Close()
}

func (a *App) Context(parent context.Context, principal cedar.EntityUID) *middleware.Context {
	return middleware.NewContext(
		parent,
		middleware.WithPrincipal(principal),
		middleware.WithStore(a.Store),
	)
}
