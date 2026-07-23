package store

import (
	"sync"

	"github.com/TheFellow/go-modular-monolith/pkg/errors"
	"github.com/mjl-/bstore"
)

var transactionLocks sync.Map

func registerTransaction(tx *bstore.Tx) {
	transactionLocks.LoadOrStore(tx, &sync.Mutex{})
}

func unregisterTransaction(tx *bstore.Tx) {
	transactionLocks.Delete(tx)
}

func LockTransaction(tx *bstore.Tx) func() {
	value, _ := transactionLocks.LoadOrStore(tx, &sync.Mutex{})
	mu := value.(*sync.Mutex)
	mu.Lock()
	return mu.Unlock
}

// Read executes f within a read transaction.
// If a transaction exists in context, uses it. Otherwise creates a new read tx.
func (s *Store) ReadContext(ctx Context, f func(*bstore.Tx) error) error {
	if tx, ok := ctx.Transaction(); ok && tx != nil {
		return f(tx)
	}
	return s.Read(ctx, f)
}

// Write executes f within the existing write transaction.
// Requires a transaction in context (set by UnitOfWork middleware).
func Write(ctx Context, f func(*bstore.Tx) error) error {
	tx, ok := ctx.Transaction()
	if !ok || tx == nil {
		return errors.Internalf("missing transaction")
	}
	return f(tx)
}
