package middleware

import (
	"time"

	"github.com/TheFellow/go-modular-monolith/pkg/authz"
	"github.com/TheFellow/go-modular-monolith/pkg/errors"
	"github.com/TheFellow/go-modular-monolith/pkg/log"
	cedar "github.com/cedar-policy/cedar-go"
)

func QueryAuthZ() QueryMiddleware {
	return func(ctx *Context, action cedar.EntityUID, next QueryNext) error {
		start := time.Now()
		err := authz.Authorize(ctx, ctx.Principal(), action)
		duration := time.Since(start)

		if mc, ok := MetricsCollectorFromContext(ctx.Context); ok && mc != nil {
			mc.RecordAuthZ(action, duration, err)
		}

		logger := log.FromContext(ctx).With(log.Args(
			log.Actor(ctx.Principal()),
			log.Action(action),
			log.Allowed(err == nil),
			log.Duration(duration),
		)...)
		switch {
		case err == nil:
			logger.Debug("authorization allowed")
		case errors.IsPermission(err):
			logger.Info("authorization denied", log.Args(log.Err(err))...)
		default:
			logger.Warn("authorization error", log.Args(log.Err(err))...)
		}

		if err != nil {
			return err
		}
		return next(ctx)
	}
}

func QueryAuthZWithResource() QueryWithResourceMiddleware {
	return func(ctx *Context, action cedar.EntityUID, resource cedar.Entity, next QueryWithResourceNext) error {
		start := time.Now()
		err := authz.AuthorizeWithEntity(ctx, ctx.Principal(), action, resource)
		duration := time.Since(start)

		if mc, ok := MetricsCollectorFromContext(ctx.Context); ok && mc != nil {
			mc.RecordAuthZ(action, duration, err)
		}

		logger := log.FromContext(ctx).With(log.Args(
			log.Actor(ctx.Principal()),
			log.Action(action),
			log.Resource(resource.UID),
			log.Allowed(err == nil),
			log.Duration(duration),
		)...)
		switch {
		case err == nil:
			logger.Debug("authorization allowed")
		case errors.IsPermission(err):
			logger.Info("authorization denied", log.Args(log.Err(err))...)
		default:
			logger.Warn("authorization error", log.Args(log.Err(err))...)
		}

		if err != nil {
			return err
		}
		return next(ctx)
	}
}

func CommandAuthZ() CommandMiddleware {
	return func(ctx *Context, action cedar.EntityUID, resource cedar.Entity, next CommandNext) error {
		start := time.Now()
		err := authz.AuthorizeWithEntity(ctx, ctx.Principal(), action, resource)
		duration := time.Since(start)

		if mc, ok := MetricsCollectorFromContext(ctx.Context); ok && mc != nil {
			mc.RecordAuthZ(action, duration, err)
		}

		logger := log.FromContext(ctx).With(log.Args(
			log.Actor(ctx.Principal()),
			log.Action(action),
			log.Resource(resource.UID),
			log.Allowed(err == nil),
			log.Duration(duration),
		)...)
		switch {
		case err == nil:
			logger.Debug("authorization allowed")
		case errors.IsPermission(err):
			logger.Info("authorization denied", log.Args(log.Err(err))...)
		default:
			logger.Warn("authorization error", log.Args(log.Err(err))...)
		}

		if err != nil {
			return err
		}
		return next(ctx)
	}
}
