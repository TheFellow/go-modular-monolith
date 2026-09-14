package middleware

import (
	"context"
	"slices"

	"github.com/TheFellow/go-modular-monolith/pkg/errors"

	cedar "github.com/cedar-policy/cedar-go"
)

type OperationKind string

const (
	OperationKindCommand OperationKind = "command"
	OperationKindQuery   OperationKind = "query"
)

type Operation struct {
	Kind   OperationKind
	Action cedar.EntityUID
}

func QueryOperation(action cedar.EntityUID) Operation {
	return Operation{
		Kind:   OperationKindQuery,
		Action: action,
	}
}

func CommandOperation(action cedar.EntityUID) Operation {
	return Operation{
		Kind:   OperationKindCommand,
		Action: action,
	}
}

type Next func(*Context) error

type Middleware func(ctx *Context, op Operation, next Next) error

type operationContextKey struct{}

type Chain struct {
	middlewares []Middleware
}

func NewChain(middlewares ...Middleware) *Chain {
	return &Chain{middlewares: middlewares}
}

func (c *Chain) Execute(ctx *Context, op Operation, final Next) error {
	// An operation inherits this marker through queries and restricted handler
	// contexts too. Rebuilding a Context cannot turn a reaction into a command.
	if op.Kind == OperationKindCommand && ctx.Value(operationContextKey{}) != nil {
		return errors.FailedPreconditionf("commands cannot run inside another operation; use a domain event reaction")
	}
	if op.Kind == OperationKindCommand {
		if tx, ok := ctx.Transaction(); ok {
			if err := tx.ClaimCommand(); err != nil {
				return err
			}
		}
	}
	ctx = ctx.forOperation()
	ctx.Context = context.WithValue(ctx.Context, operationContextKey{}, op.Kind)
	next := final
	for _, m := range slices.Backward(c.middlewares) {
		prev := next
		next = func(inner *Context) error {
			return m(inner, op, prev)
		}
	}
	return next(ctx)
}
