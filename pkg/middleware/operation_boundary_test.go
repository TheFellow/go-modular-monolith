package middleware_test

import (
	"testing"

	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
	"github.com/TheFellow/go-modular-monolith/pkg/testutil"
	cedar "github.com/cedar-policy/cedar-go"
)

func TestCommandCannotBeNestedThroughAnyOperationContext(t *testing.T) {
	t.Parallel()
	for _, kind := range []middleware.OperationKind{middleware.OperationKindCommand, middleware.OperationKindQuery} {
		t.Run(string(kind), func(t *testing.T) {
			t.Parallel()
			parent, _ := newTransactionTestStore(t)
			chain := middleware.NewChain()
			called := false
			err := chain.Execute(middleware.NewContext(parent), middleware.Operation{Kind: kind}, func(ctx *middleware.Context) error {
				variants := []*middleware.Context{ctx, middleware.NewContext(ctx), middleware.NewContext(middleware.NewHandlerContext(ctx))}
				for _, inner := range variants {
					err := chain.Execute(inner, middleware.CommandOperation(cedar.EntityUID{}), func(*middleware.Context) error { called = true; return nil })
					testutil.ErrorIsFailedPrecondition(t, err)
				}
				// Read-only operations remain available inside the active command.
				return chain.Execute(ctx, middleware.QueryOperation(cedar.EntityUID{}), func(query *middleware.Context) error {
					err := chain.Execute(query, middleware.CommandOperation(cedar.EntityUID{}), func(*middleware.Context) error { called = true; return nil })
					testutil.ErrorIsFailedPrecondition(t, err)
					return nil
				})
			})
			testutil.Ok(t, err)
			testutil.IsFalse(t, called)
		})
	}
}
