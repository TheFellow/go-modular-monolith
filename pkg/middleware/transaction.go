package middleware

import "github.com/TheFellow/go-modular-monolith/pkg/store"

func SerializeTransaction() Middleware {
	return func(ctx *Context, _ Operation, next Next) error {
		if tx, ok := ctx.Transaction(); ok && tx != nil {
			defer store.LockTransaction(tx)()
		}
		return next(ctx)
	}
}
