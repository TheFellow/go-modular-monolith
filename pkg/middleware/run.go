package middleware

import (
	"iter"

	"github.com/TheFellow/go-modular-monolith/pkg/authz"
	"github.com/TheFellow/go-modular-monolith/pkg/errors"
	"github.com/TheFellow/go-modular-monolith/pkg/paging"
	"github.com/TheFellow/go-modular-monolith/pkg/store"
	cedar "github.com/cedar-policy/cedar-go"
)

// PageQuery consumes an ordered sequence until it has a full page of
// entities the caller may see. Permission-denied entities are skipped without
// shortening the page; evaluation and storage errors still fail the query.
func (p *Pipeline) PageQuery[Req any, Item CedarEntity](
	ctx *Context,
	action cedar.EntityUID,
	req Req,
	pageRequest paging.Request,
	execute func(store.Context, Req, paging.Cursor) iter.Seq2[Item, error],
	cursor func(Item) paging.Cursor,
) (paging.Page[Item], error) {
	page := paging.Page[Item]{Items: []Item{}}
	if pageRequest.Limit <= 0 {
		return page, errors.Invalidf("page limit must be greater than zero")
	}

	err := p.query.Execute(ctx, QueryOperation(action), func(c *Context) error {
		for item, err := range execute(c, req, pageRequest.Cursor) {
			if err != nil {
				return err
			}

			err = authz.AuthorizeWithEntity(c.Principal(), action, item.CedarEntity())
			switch {
			case err == nil:
				if len(page.Items) == pageRequest.Limit {
					page.Next = cursor(page.Items[len(page.Items)-1])
					return nil
				}
				page.Items = append(page.Items, item)
			case errors.IsPermission(err):
				continue
			default:
				return err
			}
		}
		return nil
	})
	return page, err
}

// Query loads one entity and authorizes that entity before returning it.
func (p *Pipeline) Query[Req any, Res CedarEntity](
	ctx *Context,
	action cedar.EntityUID,
	req Req,
	execute func(store.Context, Req) (Res, error),
) (Res, error) {
	var out Res
	handle := AuthorizeEntityQuery(action, func(c *Context, req Req) (Res, error) {
		return execute(c, req)
	})

	err := p.query.Execute(ctx, QueryOperation(action), func(c *Context) error {
		res, err := handle(c, req)
		if err != nil {
			return err
		}
		out = res
		return nil
	})
	return out, err
}

// QueryResource authorizes an already-known resource before running a query.
// Keeping the request separate lets callers avoid artificial request wrappers
// whose only purpose is to implement CedarEntity.
func (p *Pipeline) QueryResource[Req, Res any](
	ctx *Context,
	action cedar.EntityUID,
	resource cedar.Entity,
	req Req,
	execute func(store.Context, Req) (Res, error),
) (Res, error) {
	var out Res
	err := p.query.Execute(ctx, QueryOperation(action), func(c *Context) error {
		if err := authz.AuthorizeWithEntity(c.Principal(), action, resource); err != nil {
			return err
		}
		res, err := execute(c, req)
		if err != nil {
			return err
		}
		out = res
		return nil
	})
	return out, err
}

// Command runs a command whose authorization resource is already available.
func (p *Pipeline) Command[In CedarEntity, Out CedarEntity](
	ctx *Context,
	action cedar.EntityUID,
	input In,
	handle CommandHandler[In, Out],
) (Out, error) {
	return p.LoadCommand(ctx, action, func(*Context) (In, error) { return input, nil }, handle)
}

// LoadCommand loads trusted authorization state inside the command transaction
// before handling the mutation.
func (p *Pipeline) LoadCommand[In CedarEntity, Out CedarEntity](
	ctx *Context,
	action cedar.EntityUID,
	load func(*Context) (In, error),
	handle CommandHandler[In, Out],
) (Out, error) {
	return p.loadCommand(ctx, action, load, handle, nil)
}

// LoadCommandActions is LoadCommand with authorization actions derived from
// the loaded state. It supports operations whose policy requirements depend on
// the exact transition, such as replacing a complete tag set.
func (p *Pipeline) LoadCommandActions[In CedarEntity, Out CedarEntity](
	ctx *Context,
	action cedar.EntityUID,
	load func(*Context) (In, error),
	handle CommandHandler[In, Out],
	authorizationActions func(In) []cedar.EntityUID,
) (Out, error) {
	return p.loadCommand(ctx, action, load, handle, authorizationActions)
}

func (p *Pipeline) loadCommand[In CedarEntity, Out CedarEntity](
	ctx *Context,
	action cedar.EntityUID,
	load func(*Context) (In, error),
	handle CommandHandler[In, Out],
	authorizationActions func(In) []cedar.EntityUID,
) (Out, error) {
	var out Out

	err := p.command.Execute(ctx, CommandOperation(action), func(c *Context) error {
		input, err := load(c)
		if err != nil {
			return err
		}
		inputEntity := input.CedarEntity()

		if activity, ok := c.Activity(); ok && activity.Resource.IsZero() {
			activity.Resource = inputEntity.UID
		}

		actions := []cedar.EntityUID{action}
		if authorizationActions != nil {
			actions = authorizationActions(input)
		}
		if len(actions) == 0 {
			return errors.Internalf("command authorization actions are required")
		}
		authorizedHandle := authorizeCommandActions(actions, handle)
		res, err := authorizedHandle(c, input)
		if err != nil {
			return err
		}

		if activity, ok := c.Activity(); ok && activity.Resource.IsZero() {
			activity.Resource = res.CedarEntity().UID
		}

		out = res
		return nil
	})
	return out, err
}

func authorizeCommandActions[In CedarEntity, Out CedarEntity](
	actions []cedar.EntityUID,
	next CommandHandler[In, Out],
) CommandHandler[In, Out] {
	return func(ctx *Context, in In) (Out, error) {
		var zero Out
		for _, action := range actions {
			if err := authz.AuthorizeWithEntity(ctx.Principal(), action, in.CedarEntity()); err != nil {
				return zero, err
			}
		}

		out, err := next(ctx, in)
		if err != nil {
			return zero, err
		}
		for _, action := range actions {
			if err := authz.AuthorizeWithEntity(ctx.Principal(), action, out.CedarEntity()); err != nil {
				return zero, err
			}
		}
		return out, nil
	}
}
