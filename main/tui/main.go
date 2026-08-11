package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/urfave/cli/v3"

	"github.com/TheFellow/go-modular-monolith/app"
	"github.com/TheFellow/go-modular-monolith/pkg/authn"
	"github.com/TheFellow/go-modular-monolith/pkg/errors"
	pkglog "github.com/TheFellow/go-modular-monolith/pkg/log"
	"github.com/TheFellow/go-modular-monolith/pkg/runtimeconfig"
	"github.com/TheFellow/go-modular-monolith/pkg/store"
	"github.com/TheFellow/go-modular-monolith/pkg/telemetry"
)

const defaultDatabasePath = runtimeconfig.DefaultDatabasePath

type tuiConfig struct {
	databasePath  string
	actor         string
	logLevel      string
	logFormat     string
	logFile       string
	enableMetrics bool
}

func main() {
	if err := newCommand().Run(context.Background(), os.Args); err != nil {
		cli.HandleExitCoder(errors.ToCLIExit(err))
		os.Exit(errors.ExitGeneral)
	}
}

func newCommand() *cli.Command {
	defaults := runtimeconfig.Default()
	config := tuiConfig{
		databasePath: defaults.DatabasePath,
		actor:        defaults.Actor,
		logLevel:     defaults.LogLevel,
		logFormat:    defaults.LogFormat,
		logFile:      defaultLogPath(defaults.DatabasePath),
	}
	return &cli.Command{
		Name:  "mixology-tui",
		Usage: "Interactive terminal client for Mixology",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "db", Value: config.databasePath, Usage: "Database path", Destination: &config.databasePath, Sources: cli.EnvVars(runtimeconfig.EnvDatabasePath)},
			&cli.StringFlag{Name: "log-level", Value: config.logLevel, Usage: "Log level (debug, info, warn, error)", Destination: &config.logLevel, Sources: cli.EnvVars(runtimeconfig.EnvLogLevel)},
			&cli.StringFlag{Name: "log-format", Value: config.logFormat, Usage: "Log format (text, json)", Destination: &config.logFormat, Sources: cli.EnvVars(runtimeconfig.EnvLogFormat)},
			&cli.StringFlag{Name: "log-file", Value: config.logFile, Usage: "Write logs to file", Destination: &config.logFile, Sources: cli.EnvVars(runtimeconfig.EnvLogFile)},
			&cli.StringFlag{Name: "actor", Aliases: []string{"as"}, Value: config.actor, Usage: "Actor to run as (owner|manager|sommelier|bartender|anonymous)", Destination: &config.actor, Sources: cli.EnvVars(runtimeconfig.EnvActor)},
			&cli.BoolFlag{Name: "metrics", Usage: "Enable Prometheus metrics endpoint on :9090/metrics", Destination: &config.enableMetrics, Sources: cli.EnvVars(runtimeconfig.EnvMetrics)},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			if !cmd.IsSet("log-file") {
				config.logFile = defaultLogPath(config.databasePath)
			}
			return run(ctx, config, config.databasePath)
		},
	}
}

func run(ctx context.Context, config tuiConfig, databasePath string) error {
	if err := os.MkdirAll(filepath.Dir(config.logFile), 0o755); err != nil {
		return fmt.Errorf("create log dir: %w", err)
	}
	logFile, err := os.OpenFile(config.logFile, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0o600)
	if err != nil {
		return fmt.Errorf("open log file: %w", err)
	}
	defer func() { _ = logFile.Close() }()

	var metrics = telemetry.Nop()
	var metricsServer *http.Server
	var shutdownMetrics func(context.Context) error
	if config.enableMetrics {
		prom, err := telemetry.NewPrometheus()
		if err != nil {
			return err
		}
		metrics = prom.Metrics
		shutdownMetrics = prom.Shutdown
		mux := http.NewServeMux()
		mux.Handle("/metrics", prom.Handler)
		metricsServer = &http.Server{Addr: runtimeconfig.DefaultMetricsAddr, Handler: mux}
		go func() { _ = metricsServer.ListenAndServe() }()
	}
	if metricsServer != nil {
		defer func() { _ = metricsServer.Shutdown(ctx) }()
	}
	if shutdownMetrics != nil {
		defer func() { _ = shutdownMetrics(ctx) }()
	}

	principal, err := authn.ParseActor(config.actor)
	if err != nil {
		return err
	}
	ctx = pkglog.ToContext(ctx, pkglog.Setup(config.logLevel, config.logFormat, logFile))
	ctx = telemetry.WithMetrics(ctx, metrics)
	ctx = authn.ToContext(ctx, principal)

	database, err := store.Open(ctx, databasePath)
	if err != nil {
		return err
	}
	application := app.New(ctx, app.Config{Store: database})
	defer func() { _ = application.Close() }()
	changes, err := database.MonitorChanges(ctx, store.DefaultChangePollInterval)
	if err != nil {
		return err
	}
	defer changes.Close()

	program := tea.NewProgram(NewApp(app.NewSession(ctx, application), changes), tea.WithAltScreen())
	_, err = program.Run()
	return err
}

func defaultLogPath(databasePath string) string {
	directory := filepath.Dir(databasePath)
	if directory == "" || directory == "." {
		return "mixology-tui.log"
	}
	return filepath.Join(directory, "mixology-tui.log")
}
