package main

import (
	"context"
	"fmt"
	"net/http"
	"os"

	"github.com/TheFellow/go-modular-monolith/app"
	"github.com/TheFellow/go-modular-monolith/pkg/authn"
	apperrors "github.com/TheFellow/go-modular-monolith/pkg/errors"
	pkglog "github.com/TheFellow/go-modular-monolith/pkg/log"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
	"github.com/TheFellow/go-modular-monolith/pkg/store"
	"github.com/TheFellow/go-modular-monolith/pkg/telemetry"
	"github.com/urfave/cli/v3"
)

type CLI struct {
	app             *app.App
	dbPath          string
	actor           string
	logLevel        string
	logFormat       string
	enableMetrics   bool
	metricsServer   *http.Server
	metricsShutdown func(context.Context) error
}

func NewCLI() (*CLI, error) {
	return &CLI{
		dbPath:    "data/mixology.db",
		actor:     "owner",
		logLevel:  "info",
		logFormat: "text",
	}, nil
}

func (c *CLI) action(fn func(*middleware.Context, *cli.Command) error) cli.ActionFunc {
	return func(ctx context.Context, cmd *cli.Command) error {
		mctx, ok := ctx.(*middleware.Context)
		if !ok {
			return apperrors.ToCLIExit(fmt.Errorf("expected middleware context"))
		}
		return apperrors.ToCLIExit(fn(mctx, cmd))
	}
}

func (c *CLI) Command() *cli.Command {
	return &cli.Command{
		Name:  "mixology",
		Usage: "Mixology as a Service",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:        "log-level",
				Value:       c.logLevel,
				Usage:       "Log level (debug, info, warn, error)",
				Destination: &c.logLevel,
				Sources:     cli.EnvVars("MIXOLOGY_LOG_LEVEL"),
			},
			&cli.StringFlag{
				Name:        "log-format",
				Value:       c.logFormat,
				Usage:       "Log format (text, json)",
				Destination: &c.logFormat,
				Sources:     cli.EnvVars("MIXOLOGY_LOG_FORMAT"),
			},
			&cli.StringFlag{
				Name:        "as",
				Usage:       "Actor to run as (owner|anonymous)",
				Value:       c.actor,
				Destination: &c.actor,
			},
			&cli.BoolFlag{
				Name:        "metrics",
				Usage:       "Enable Prometheus metrics endpoint on :9090/metrics",
				Destination: &c.enableMetrics,
				Sources:     cli.EnvVars("MIXOLOGY_METRICS"),
			},
		},
		Before: func(ctx context.Context, _ *cli.Command) (context.Context, error) {
			logger := pkglog.Setup(c.logLevel, c.logFormat, os.Stderr)

			var metrics telemetry.Metrics = telemetry.Nop()
			if c.enableMetrics {
				prom, err := telemetry.NewPrometheus()
				if err != nil {
					return ctx, err
				}
				metrics = prom.Metrics
				c.metricsShutdown = prom.Shutdown

				mux := http.NewServeMux()
				mux.Handle("/metrics", prom.Handler)
				c.metricsServer = &http.Server{Addr: ":9090", Handler: mux}
				go func() { _ = c.metricsServer.ListenAndServe() }()
			}

			s, err := store.Open(c.dbPath)
			if err != nil {
				return ctx, err
			}
			c.app = app.New(
				app.WithStore(s),
				app.WithLogger(logger),
				app.WithMetrics(metrics),
			)

			p, err := authn.ParseActor(c.actor)
			if err != nil {
				return ctx, err
			}
			return c.app.Context(ctx, p), nil
		},
		After: func(ctx context.Context, _ *cli.Command) error {
			if c.app != nil {
				_ = c.app.Close()
			}
			if c.metricsServer != nil {
				_ = c.metricsServer.Shutdown(ctx)
			}
			if c.metricsShutdown != nil {
				_ = c.metricsShutdown(ctx)
			}
			return nil
		},
		ExitErrHandler: func(_ context.Context, _ *cli.Command, _ error) {},
		OnUsageError: func(_ context.Context, _ *cli.Command, err error, _ bool) error {
			return cli.Exit(err, apperrors.ExitUsage)
		},
			Commands: []*cli.Command{
				c.drinksCommands(),
				c.ingredientsCommands(),
				c.inventoryCommands(),
				c.menuCommands(),
				c.ordersCommands(),
				c.auditCommands(),
			},
		}
	}
