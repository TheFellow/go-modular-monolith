package app

import (
	"github.com/TheFellow/go-modular-monolith/pkg/store"
)

func Open(path string) (*App, error) {
	s, err := store.Open(path)
	if err != nil {
		return nil, err
	}
	return New(WithStore(s)), nil
}
