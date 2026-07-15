package store

import (
	"context"

	"github.com/mjl-/bstore"
)

// Context exposes transaction state to persistence operations.
type Context interface {
	context.Context
	Transaction() (*bstore.Tx, bool)
}
