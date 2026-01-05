package middleware

import (
	"time"

	cedar "github.com/cedar-policy/cedar-go"

	"github.com/TheFellow/go-modular-monolith/pkg/log"
)

func CommandLogging() CommandMiddleware {
	return func(ctx *Context, action cedar.EntityUID, resource cedar.Entity, next CommandNext) error {
		logger := log.FromContext(ctx).With(log.Args(
			log.Actor(ctx.Principal()),
			log.Action(action),
			log.Resource(resource.UID),
		)...)

		start := time.Now()
		logger.Debug("command started")

		err := next(ctx)

		if err != nil {
			logger.Error("command failed", log.Args(log.Duration(time.Since(start)), log.Err(err))...)
		} else {
			logger.Info("command completed", log.Args(log.Duration(time.Since(start)))...)
		}
		return err
	}
}

func QueryLogging() QueryMiddleware {
	return func(ctx *Context, action cedar.EntityUID, next QueryNext) error {
		logger := log.FromContext(ctx).With(log.Args(
			log.Actor(ctx.Principal()),
			log.Action(action),
		)...)

		start := time.Now()
		logger.Debug("query started")

		err := next(ctx)

		if err != nil {
			logger.Warn("query failed", log.Args(log.Duration(time.Since(start)), log.Err(err))...)
		} else {
			logger.Debug("query completed", log.Args(log.Duration(time.Since(start)))...)
		}
		return err
	}
}

func QueryWithResourceLogging() QueryWithResourceMiddleware {
	return func(ctx *Context, action cedar.EntityUID, resource cedar.Entity, next QueryWithResourceNext) error {
		logger := log.FromContext(ctx).With(log.Args(
			log.Actor(ctx.Principal()),
			log.Action(action),
			log.Resource(resource.UID),
		)...)

		start := time.Now()
		logger.Debug("query started")

		err := next(ctx)

		if err != nil {
			logger.Warn("query failed", log.Args(log.Duration(time.Since(start)), log.Err(err))...)
		} else {
			logger.Debug("query completed", log.Args(log.Duration(time.Since(start)))...)
		}
		return err
	}
}
