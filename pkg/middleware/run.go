package middleware

import (
	"github.com/TheFellow/go-modular-monolith/pkg/store"
	cedar "github.com/cedar-policy/cedar-go"
)

// RunListQuery loads a list and elides entities the caller cannot see.
func RunListQuery[Req any, Item CedarEntity](
	pipeline *Pipeline,
	ctx *Context,
	action cedar.EntityUID,
	execute func(store.Context, Req) ([]Item, error),
	req Req,
) ([]Item, error) {
	var out []Item
	handle := AuthorizeListQuery(action, func(c *Context, req Req) ([]Item, error) {
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
