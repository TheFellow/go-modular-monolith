package middleware

import (
	"github.com/TheFellow/go-modular-monolith/pkg/errors"
	cedar "github.com/cedar-policy/cedar-go"
)

func RunQuery[Req, Res any](
	ctx *Context,
	action cedar.EntityUID,
	execute func(*Context, Req) (Res, error),
	req Req,
) (Res, error) {
	var out Res

	err := Query.Execute(ctx, action, func(c *Context) error {
		res, err := execute(c, req)
		if err != nil {
			return err
		}
		out = res
		return nil
	})
	return out, err
}

func RunQueryWithResource[Req CedarEntity, Res any](
	ctx *Context,
	action cedar.EntityUID,
	execute func(*Context, Req) (Res, error),
	req Req,
) (Res, error) {
	var out Res

	resource := req.CedarEntity()
	err := QueryWithResource.Execute(ctx, action, resource, func(c *Context) error {
		res, err := execute(c, req)
		if err != nil {
			return err
		}
		out = res
		return nil
	})
	return out, err
}

func RunCommand[In CedarEntity, Out CedarEntity](
	ctx *Context,
	action cedar.EntityUID,
	load func(*Context) (In, error),
	execute func(*Context, In) (Out, error),
) (Out, error) {
	var out Out

	WithCommandLoader(func(c *Context) (CedarEntity, error) {
		return load(c)
	})(ctx)

	err := Command.Execute(ctx, action, cedar.Entity{}, func(c *Context) error {
		input, ok := c.InputEntity()
		if !ok {
			return errors.Internalf("command input missing")
		}
		typedInput, ok := input.(In)
		if !ok {
			return errors.Internalf("command input type mismatch")
		}
		res, err := execute(c, typedInput)
		if err != nil {
			return err
		}
		c.setOutputEntity(res)
		out = res
		return nil
	})
	return out, err
}
