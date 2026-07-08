package store

import (
	"context"

	"github.com/mjl-/bstore"
)

// Context provides data access capabilities.
// *middleware.Context implements this interface.
type Context interface {
	context.Context
	Store() (*Store, bool)
	Transaction() (*bstore.Tx, bool)
}
