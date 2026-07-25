package commands

import (
	"github.com/TheFellow/go-modular-monolith/app/domains/ingredients/internal/dao"
	"github.com/TheFellow/go-modular-monolith/app/domains/tagging"
	"github.com/TheFellow/go-modular-monolith/pkg/store"
)

type Commands struct {
	dao *dao.DAO
}

func New(s *store.Store, tags *tagging.Repository) *Commands {
	return &Commands{dao: dao.New(s, tags)}
}
