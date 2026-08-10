package store

import (
	"sync"

	"github.com/TheFellow/go-modular-monolith/pkg/errors"
)

var transactionLocks sync.Map

func registerTransaction(tx *Tx)   { transactionLocks.LoadOrStore(tx, &sync.Mutex{}) }
func unregisterTransaction(tx *Tx) { transactionLocks.Delete(tx) }

func LockTransaction(tx *Tx) func() {
	value, _ := transactionLocks.LoadOrStore(tx, &sync.Mutex{})
	mu := value.(*sync.Mutex)
	mu.Lock()
	return mu.Unlock
}

func (s *Store) ReadContext(ctx Context, f func(*Tx) error) error {
	if tx, ok := ctx.Transaction(); ok && tx != nil {
		return f(tx)
	}
	return s.Read(ctx, f)
}

func Write(ctx Context, f func(*Tx) error) error {
	tx, ok := ctx.Transaction()
	if !ok || tx == nil {
		return errors.Internalf("missing transaction")
	}
	return f(tx)
}
