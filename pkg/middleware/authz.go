package middleware

import (
	"github.com/TheFellow/go-modular-monolith/pkg/authz"
	"github.com/TheFellow/go-modular-monolith/pkg/errors"
	cedar "github.com/cedar-policy/cedar-go"
)

// QueryHandler executes a query and returns its result.
type QueryHandler[Req, Out any] func(*Context, Req) (Out, error)

// AuthorizeListQuery filters a loaded list to the entities the caller is
// authorized to see. A permission denial hides that entity; evaluation and
// infrastructure errors still fail the query.
func AuthorizeListQuery[Req any, Item CedarEntity](action cedar.EntityUID, next QueryHandler[Req, []Item]) QueryHandler[Req, []Item] {
	return func(ctx *Context, req Req) ([]Item, error) {
		items, err := next(ctx, req)
		if err != nil {
			return nil, err
		}

		visible := make([]Item, 0, len(items))
		for _, item := range items {
			err := authz.AuthorizeWithEntity(ctx.Principal(), action, item.CedarEntity())
			switch {
			case err == nil:
				visible = append(visible, item)
			case errors.IsPermission(err):
				continue
			default:
				return nil, err
			}
		}
		return visible, nil
	}
}

// AuthorizeEntityQuery executes a query and authorizes its loaded result
// before returning it to the caller. This is the authorization shape used by
// get queries.
func AuthorizeEntityQuery[Req any, Out CedarEntity](action cedar.EntityUID, next QueryHandler[Req, Out]) QueryHandler[Req, Out] {
	return func(ctx *Context, req Req) (Out, error) {
		var zero Out
		out, err := next(ctx, req)
		if err != nil {
			return zero, err
		}
		if err := authz.AuthorizeWithEntity(ctx.Principal(), action, out.CedarEntity()); err != nil {
			return zero, err
		}
		return out, nil
	}
}

// CommandHandler handles a loaded command resource and returns its resulting
// state.
type CommandHandler[In CedarEntity, Out CedarEntity] func(*Context, In) (Out, error)

// AuthorizeCommand authorizes both sides of a command state transition. The
// loaded resource is checked before the handler runs, and the resulting
// resource is checked before the result is returned.
func AuthorizeCommand[In CedarEntity, Out CedarEntity](action cedar.EntityUID, next CommandHandler[In, Out]) CommandHandler[In, Out] {
	return func(ctx *Context, in In) (Out, error) {
		var zero Out
		if err := authz.AuthorizeWithEntity(ctx.Principal(), action, in.CedarEntity()); err != nil {
			return zero, err
		}

		out, err := next(ctx, in)
		if err != nil {
			return zero, err
		}
		if err := authz.AuthorizeWithEntity(ctx.Principal(), action, out.CedarEntity()); err != nil {
			return zero, err
		}
		return out, nil
	}
}
