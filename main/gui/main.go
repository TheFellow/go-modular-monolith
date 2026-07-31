package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"

	"github.com/TheFellow/go-modular-monolith/pkg/authn"
)

const defaultDatabasePath = "data/mixology.db"

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
	dataDirectory, err := defaultDataDirectory()
	if err != nil {
		return nil, err
	}
	config := desktopConfig{
		dataDirectory: dataDirectory,
		databasePath:  defaultDatabasePath,
		actor:         "owner",
	}
	flags := flag.NewFlagSet("mixology-fyne", flag.ContinueOnError)
	flags.SetOutput(output)
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
