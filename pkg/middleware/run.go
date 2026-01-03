package middleware

import (
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

func RunCommand[Req, Res any](
	ctx *Context,
	action cedar.EntityUID,
	resource cedar.Entity,
	execute func(*Context, Req) (Res, error),
	req Req,
) (Res, error) {
	var out Res

	err := Command.Execute(ctx, action, resource, func(c *Context) error {
		res, err := execute(c, req)
		if err != nil {
			return err
		}
		out = res
		return nil
	})
	return out, err
}
