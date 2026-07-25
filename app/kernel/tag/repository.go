package tag

import (
	"github.com/TheFellow/go-modular-monolith/pkg/store"
	cedar "github.com/cedar-policy/cedar-go"
	"github.com/mjl-/bstore"
)

// Repository is the cross-domain read and lifecycle boundary used by entity owners.
type Repository interface {
	ListTypeTx(*bstore.Tx, cedar.EntityType, []cedar.String) (map[cedar.EntityUID]Tags, error)
	DeleteTarget(store.Context, cedar.EntityUID) (int, error)
}
