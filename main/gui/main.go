package main

import (
	"context"
	"flag"
	"fmt"
	"github.com/TheFellow/go-modular-monolith/pkg/errors"
	"io"
	"os"
	"strconv"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"

	"github.com/TheFellow/go-modular-monolith/pkg/authn"
	"github.com/TheFellow/go-modular-monolith/pkg/runtimeconfig"
)

const defaultDatabasePath = runtimeconfig.DefaultDatabasePath

func main() {
	config, err := startupConfig(os.Args[1:], os.Stderr)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if config == nil {
		return
	}
	gui := app.NewWithID(applicationID)
	app.SetMetadata(withFyneDoMigration(gui.Metadata()))
	desktop, err := openDesktop(context.Background(), gui, *config)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	defer func() { _ = desktop.Close() }()
	desktop.window.ShowAndRun()
}

func withFyneDoMigration(metadata fyne.AppMetadata) fyne.AppMetadata {
	if metadata.Migrations == nil {
		metadata.Migrations = make(map[string]bool)
	}
	metadata.Migrations["fyneDo"] = true
	return metadata
}

func startupConfig(args []string, output io.Writer) (*desktopConfig, error) {
	defaults := runtimeconfig.Default()
	enableMetrics, err := environmentBool(runtimeconfig.EnvMetrics)
	if err != nil {
		return nil, err
	}
	dataDirectory, err := defaultDataDirectory()
	if err != nil {
		return nil, err
	}
	config := desktopConfig{
		dataDirectory: environmentOr(runtimeconfig.EnvDataDir, dataDirectory),
		databasePath:  environmentOr(runtimeconfig.EnvDatabasePath, defaults.DatabasePath),
		actor:         environmentOr(runtimeconfig.EnvActor, defaults.Actor),
		logLevel:      environmentOr(runtimeconfig.EnvLogLevel, defaults.LogLevel),
		logFormat:     environmentOr(runtimeconfig.EnvLogFormat, defaults.LogFormat),
		enableMetrics: enableMetrics,
	}
	flags := flag.NewFlagSet("mixology-fyne", flag.ContinueOnError)
	flags.SetOutput(output)
	flags.StringVar(&config.dataDirectory, "data-dir", config.dataDirectory, "application data and default log directory (or "+runtimeconfig.EnvDataDir+")")
	flags.StringVar(&config.databasePath, "db", config.databasePath, "database path (or "+runtimeconfig.EnvDatabasePath+")")
	flags.StringVar(&config.logLevel, "log-level", config.logLevel, "log level (debug, info, warn, error)")
	flags.StringVar(&config.logFormat, "log-format", config.logFormat, "log format (text, json)")
	flags.StringVar(&config.logFile, "log-file", environmentOr(runtimeconfig.EnvLogFile, config.logFile), "diagnostic log path (or "+runtimeconfig.EnvLogFile+")")
	flags.BoolVar(&config.enableMetrics, "metrics", config.enableMetrics, "enable Prometheus metrics on "+runtimeconfig.DefaultMetricsAddr+"/metrics")
	flags.StringVar(&config.actor, "actor", config.actor, "actor to run as (owner|manager|sommelier|bartender|anonymous)")
	flags.StringVar(&config.actor, "as", config.actor, "alias for -actor")
	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil, nil
		}
		return nil, err
	}
	if flags.NArg() != 0 {
		return nil, fmt.Errorf("unexpected arguments: %v", flags.Args())
	}
	if _, err := authn.ParseActor(config.actor); err != nil {
		return nil, err
	}
	return &config, nil
}

func environmentOr(name, fallback string) string {
	if value, ok := os.LookupEnv(name); ok {
		return value
	}
	return fallback
}

func environmentBool(name string) (bool, error) {
	value, ok := os.LookupEnv(name)
	if !ok || value == "" {
		return false, nil
	}
	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return false, fmt.Errorf("parse %s: %w", name, err)
	}
	return parsed, nil
}
