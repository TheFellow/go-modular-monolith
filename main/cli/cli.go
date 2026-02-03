package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"github.com/TheFellow/go-modular-monolith/app"
	"github.com/TheFellow/go-modular-monolith/main/tui"
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
	logFile         string
	logFileHandle   *os.File
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
			&cli.BoolFlag{
				Name:  "tui",
				Usage: "Launch interactive terminal UI",
			},
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
				Name:        "log-file",
				Usage:       "Write logs to file instead of stderr",
				Destination: &c.logFile,
				Sources:     cli.EnvVars("MIXOLOGY_LOG_FILE"),
			},
			&cli.StringFlag{
				Name:        "actor",
				Aliases:     []string{"as"},
				Usage:       "Actor to run as (owner|manager|sommelier|bartender|anonymous)",
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
		Before: func(ctx context.Context, cmd *cli.Command) (context.Context, error) {
			if cmd != nil && cmd.Bool("tui") && c.logFile == "" {
				logDir := filepath.Dir(c.dbPath)
				if logDir != "" && logDir != "." {
					if err := os.MkdirAll(logDir, 0o755); err != nil {
						return ctx, fmt.Errorf("create log dir: %w", err)
					}
					c.logFile = filepath.Join(logDir, "mixology-tui.log")
				} else {
					c.logFile = "mixology-tui.log"
				}
			}

			var logOutput io.Writer = os.Stderr
			if c.logFile != "" {
				f, err := os.OpenFile(c.logFile, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0o600)
				if err != nil {
					return ctx, fmt.Errorf("open log file: %w", err)
				}
				logOutput = f
				c.logFileHandle = f
			}
			logger := pkglog.Setup(c.logLevel, c.logFormat, logOutput)

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

			ctx = c.app.Context(ctx, p)
			mctx, ok := ctx.(*middleware.Context)

			if cmd != nil && cmd.Bool("tui") {
				if !ok {
					return ctx, cli.Exit(fmt.Errorf("expected middleware context for TUI"), apperrors.ExitGeneral)
				}
				initialView := tui.ViewDashboard
				args := cmd.Args().Slice()
				if len(args) > 0 {
					var ok bool
					initialView, ok = tui.ParseView(args[0])
					if !ok {
						return ctx, cli.Exit(fmt.Errorf("unknown view: %s", args[0]), apperrors.ExitUsage)
					}
				}
				if len(args) > 1 {
					return ctx, cli.Exit(fmt.Errorf("too many arguments for --tui"), apperrors.ExitUsage)
				}

				if err := tui.Run(p, c.app, initialView); err != nil {
					return ctx, err
				}
				return ctx, cli.Exit("", 0)
			}

			if ok {
				return mctx, nil
			}
			return ctx, nil
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
			if c.logFileHandle != nil {
				_ = c.logFileHandle.Close()
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
