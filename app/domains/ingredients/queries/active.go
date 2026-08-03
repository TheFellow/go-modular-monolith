package queries

import (
	"github.com/TheFellow/go-modular-monolith/pkg/store"
	cedar "github.com/cedar-policy/cedar-go"
)

func (q *Queries) ActiveIDs(ctx store.Context, ids []cedar.String) (map[cedar.String]struct{}, error) {
	return q.dao.ActiveIDs(ctx, ids)
}
