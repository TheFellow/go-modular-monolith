package middleware

import (
	"slices"

	cedar "github.com/cedar-policy/cedar-go"
)

type OperationKind string

const (
	OperationKindCommand OperationKind = "command"
	OperationKindQuery   OperationKind = "query"
)

type Operation struct {
	Kind        OperationKind
	Action      cedar.EntityUID
	Resource    cedar.Entity
	hasResource bool
}

func QueryOperation(action cedar.EntityUID) Operation {
	return Operation{
		Kind:   OperationKindQuery,
		Action: action,
	}
}

func QueryResourceOperation(action cedar.EntityUID, resource cedar.Entity) Operation {
	return Operation{
		Kind:        OperationKindQuery,
		Action:      action,
		Resource:    resource,
		hasResource: true,
	}
}

func CommandOperation(action cedar.EntityUID) Operation {
	return Operation{
		Kind:   OperationKindCommand,
		Action: action,
	}
}

func (op Operation) HasResource() bool {
	return op.hasResource
}

type Next func(*Context) error

type Middleware func(ctx *Context, op Operation, next Next) error

type Chain struct {
	middlewares []Middleware
}

func NewChain(middlewares ...Middleware) *Chain {
	return &Chain{middlewares: middlewares}
}

func (c *Chain) Execute(ctx *Context, op Operation, final Next) error {
	next := final
	for _, m := range slices.Backward(c.middlewares) {
		prev := next
		next = func(inner *Context) error {
			return m(inner, op, prev)
		}
	}
	return next(ctx)
}
