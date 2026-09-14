package tagging

import (
	"context"

	"github.com/TheFellow/go-modular-monolith/app/domains/tagging/internal/dao"
	"github.com/TheFellow/go-modular-monolith/pkg/store"
)

type Repository = dao.Repository

func NewRepository(s *store.Store) *Repository           { return dao.NewRepository(s) }
func RegisterSchema(ctx context.Context, s *store.Store) { dao.RegisterSchema(ctx, s) }
