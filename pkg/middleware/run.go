package middleware

import (
	"iter"

	"github.com/TheFellow/go-modular-monolith/pkg/authz"
	"github.com/TheFellow/go-modular-monolith/pkg/errors"
	"github.com/TheFellow/go-modular-monolith/pkg/paging"
	"github.com/TheFellow/go-modular-monolith/pkg/store"
	cedar "github.com/cedar-policy/cedar-go"
)

// RunPageQuery consumes an ordered sequence until it has a full page of
// entities the caller may see. Permission-denied entities are skipped without
// shortening the page; evaluation and storage errors still fail the query.
func RunPageQuery[Req any, Item CedarEntity](
	pipeline *Pipeline,
	ctx *Context,
	action cedar.EntityUID,
	execute func(store.Context, Req, paging.Cursor) iter.Seq2[Item, error],
	cursor func(Item) paging.Cursor,
	req Req,
	pageRequest paging.Request,
) (paging.Page[Item], error) {
	page := paging.Page[Item]{Items: []Item{}}
	if pageRequest.Limit <= 0 {
		return page, errors.Invalidf("page limit must be greater than zero")
	}

	err := pipeline.query.Execute(ctx, QueryOperation(action), func(c *Context) error {
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

// RunEntityQuery loads one entity and authorizes that entity before returning
// it to the caller.
func RunEntityQuery[Req any, Res CedarEntity](
	pipeline *Pipeline,
	ctx *Context,
	action cedar.EntityUID,
	execute func(store.Context, Req) (Res, error),
	req Req,
) (Res, error) {
	var out Res
	handle := AuthorizeEntityQuery(action, func(c *Context, req Req) (Res, error) {
		return execute(c, req)
	})

	err := pipeline.query.Execute(ctx, QueryOperation(action), func(c *Context) error {
		res, err := handle(c, req)
		if err != nil {
			return err
		}
		out = res
		return nil
	})
	return out, err
}

// CommandSpec names the command orchestration steps RunCommand performs.
type CommandSpec[In CedarEntity, Out CedarEntity] struct {
	Action cedar.EntityUID
	Load   func(*Context) (In, error)
	Handle CommandHandler[In, Out]
}

func RunCommand[In CedarEntity, Out CedarEntity](pipeline *Pipeline, ctx *Context, spec CommandSpec[In, Out]) (Out, error) {
	var out Out

	err := pipeline.command.Execute(ctx, CommandOperation(spec.Action), func(c *Context) error {
		input, err := spec.Load(c)
		if err != nil {
			return err
		}
		inputEntity := input.CedarEntity()

		if activity, ok := c.Activity(); ok && activity.Resource.IsZero() {
			activity.Resource = inputEntity.UID
		}

		handle := AuthorizeCommand(spec.Action, spec.Handle)
		res, err := handle(c, input)
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
