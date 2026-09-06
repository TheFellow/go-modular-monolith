package queries

import (
	"github.com/TheFellow/go-modular-monolith/app/domains/menus/internal/availability"
	"github.com/TheFellow/go-modular-monolith/app/domains/menus/internal/dao"
	"github.com/TheFellow/go-modular-monolith/app/kernel/tag"
	"github.com/TheFellow/go-modular-monolith/pkg/store"
)

type Queries struct {
	store        *store.Store
	tags         tag.Repository
	dao          *dao.DAO
	availability *availability.AvailabilityCalculator
}

func New(s *store.Store, tags tag.Repository) *Queries {
	return &Queries{store: s, tags: tags, dao: dao.New(s, tags), availability: availability.New(s, tags)}
}
