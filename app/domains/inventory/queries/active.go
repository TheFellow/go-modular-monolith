package queries

import (
	"github.com/TheFellow/go-modular-monolith/pkg/set"
	"github.com/TheFellow/go-modular-monolith/pkg/store"
	cedar "github.com/cedar-policy/cedar-go"
)

func (q *Queries) ActiveIDs(ctx store.Context, ids []cedar.String) (set.Set[cedar.String], error) {
	return q.dao.ActiveIDs(ctx, ids)
}
