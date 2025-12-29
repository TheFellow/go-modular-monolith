package middleware

import (
	"github.com/TheFellow/go-modular-monolith/pkg/uow"
	cedar "github.com/cedar-policy/cedar-go"
)

func UnitOfWork(m *uow.Manager) CommandMiddleware {
	return func(ctx *Context, _ cedar.EntityUID, _ cedar.Entity, next CommandNext) error {
		tx, err := m.Begin(ctx)
		if err != nil {
			return err
		}
		ctx.SetUnitOfWork(tx)

		if err := next(ctx); err != nil {
			tx.Rollback()
			return err
		}

		return tx.Commit()
	}
}
