package main

import (
	"bytes"
	"strings"
	"testing"

	"fyne.io/fyne/v2"
	"github.com/TheFellow/go-modular-monolith/pkg/testutil"
)

func TestFyneDoMigrationPreservesMetadata(t *testing.T) {
	t.Parallel()
	metadata := withFyneDoMigration(fyne.AppMetadata{
		ID:         applicationID,
		Name:       "Mixology",
		Migrations: map[string]bool{"other": true},
	})
	if metadata.ID != applicationID || metadata.Name != "Mixology" {
		testutil.ErrorIf(t, true, "application metadata was not preserved: %#v", metadata)
	}
	if !metadata.Migrations["fyneDo"] || !metadata.Migrations["other"] {
		testutil.ErrorIf(t, true, "migration metadata = %#v", metadata.Migrations)
	}
}

func TestFyneDoMigrationInitializesMissingMigrationMetadata(t *testing.T) {
	t.Parallel()
	metadata := withFyneDoMigration(fyne.AppMetadata{})
	if !metadata.Migrations["fyneDo"] {
		testutil.ErrorIf(t, true, "migration metadata = %#v", metadata.Migrations)
	}
}

func TestStartupConfigSelectsEverySupportedActor(t *testing.T) {
	t.Parallel()
	for _, actor := range []string{"owner", "manager", "sommelier", "bartender", "anonymous"} {
		t.Run(actor, func(t *testing.T) {
			t.Parallel()
			config, err := startupConfig([]string{"-actor", actor}, new(bytes.Buffer))
			if err != nil || config == nil || config.actor != actor || config.dataDirectory == "" || config.databasePath != defaultDatabasePath {
				testutil.ErrorIf(t, true, "startup config = %#v, %v", config, err)
			}
		})
	}
}

func TestStartupConfigDefaultsFreshDesktopToOwnerAndSupportsAlias(t *testing.T) {
	t.Parallel()
	config, err := startupConfig(nil, new(bytes.Buffer))
	if err != nil || config == nil || config.actor != "owner" || config.databasePath != defaultDatabasePath {
		testutil.ErrorIf(t, true, "default startup config = %#v, %v", config, err)
	}
	config, err = startupConfig([]string{"-as", "bartender"}, new(bytes.Buffer))
	if err != nil || config == nil || config.actor != "bartender" {
		testutil.ErrorIf(t, true, "alias startup config = %#v, %v", config, err)
	}
}

func TestStartupConfigHelpAndInvalidActor(t *testing.T) {
	t.Parallel()
	output := new(bytes.Buffer)
	config, err := startupConfig([]string{"-help"}, output)
	if err != nil || config != nil || !strings.Contains(output.String(), "owner|manager|sommelier|bartender|anonymous") {
		testutil.ErrorIf(t, true, "help = %#v, %v, %q", config, err, output.String())
	}
	if _, err := startupConfig([]string{"-actor", "intruder"}, new(bytes.Buffer)); err == nil {
		testutil.ErrorIf(t, true, "%v", "invalid actor accepted")
	}
}
