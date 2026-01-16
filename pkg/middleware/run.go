package middleware

import (
	"context"

	"github.com/TheFellow/go-modular-monolith/pkg/authz"
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

	err := Command.Execute(ctx, action, func(c *Context) error {
		input, err := load(c)
		if err != nil {
			return err
		}

		if activity, ok := ActivityFromContext(c.Context); ok && activity.Resource.IsZero() {
			activity.Resource = input.CedarEntity().UID
		}

		if err := authz.AuthorizeWithEntity(c.Principal(), action, input.CedarEntity()); err != nil {
			return err
		}

		res, err := execute(c, input)
		if err != nil {
			return err
		}

		if activity, ok := ActivityFromContext(c.Context); ok && activity.Resource.IsZero() {
			activity.Resource = res.CedarEntity().UID
		}

		if err := authz.AuthorizeWithEntity(c.Principal(), action, res.CedarEntity()); err != nil {
			return err
		}

		out = res
		return nil
	})
	return out, err
}

// FromModel returns a loader that yields a fixed entity (useful for Create).
func FromModel[T CedarEntity](entity T) func(*Context) (T, error) {
	return func(*Context) (T, error) {
		return entity, nil
	}
}

// ByID returns a loader that fetches an entity by ID (useful for Update/Delete).
func ByID[T CedarEntity](id cedar.EntityUID, get func(context.Context, cedar.EntityUID) (T, error)) func(*Context) (T, error) {
	return func(ctx *Context) (T, error) {
		return get(ctx, id)
	}
}
