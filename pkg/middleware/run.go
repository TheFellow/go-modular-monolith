package middleware

import (
	"context"

	cedar "github.com/cedar-policy/cedar-go"
)

func RunQuery[Req, Res any](
	ctx context.Context,
	action cedar.EntityUID,
	execute func(*Context, Req) (Res, error),
	req Req,
) (Res, error) {
	mctx := NewContext(ctx)
	var out Res

	err := Query.Execute(mctx, action, func(c *Context) error {
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
	ctx context.Context,
	action cedar.EntityUID,
	resource cedar.Entity,
	execute func(*Context, Req) (Res, error),
	req Req,
) (Res, error) {
	mctx := NewContext(ctx)
	var out Res

	err := Command.Execute(mctx, action, resource, func(c *Context) error {
		res, err := execute(c, req)
		if err != nil {
			return err
		}
		out = res
		return nil
	})
	return out, err
}
