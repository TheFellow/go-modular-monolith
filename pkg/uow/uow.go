package uow

import (
	"context"
	"fmt"
)

type Saver interface {
	Save(ctx context.Context) error
}

type Manager struct{}

func NewManager() *Manager {
	return &Manager{}
}

func (m *Manager) Begin(ctx context.Context) (*Tx, error) {
	_ = m
	return &Tx{ctx: ctx}, nil
}

type Tx struct {
	ctx    context.Context
	savers []Saver
	done   bool
}

func (t *Tx) Register(s Saver) error {
	if t.done {
		return fmt.Errorf("transaction already finished")
	}
	t.savers = append(t.savers, s)
	return nil
}

func (t *Tx) Commit() error {
	if t.done {
		return fmt.Errorf("transaction already finished")
	}
	t.done = true

	for _, s := range t.savers {
		if err := s.Save(t.ctx); err != nil {
			return err
		}
	}
	return nil
}

func (t *Tx) Rollback() {
	t.done = true
}
