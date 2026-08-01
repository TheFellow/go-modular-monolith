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
	testutil.ErrorIf(t, metadata.ID != applicationID || metadata.Name != "Mixology", "application metadata was not preserved: %#v", metadata)
	testutil.ErrorIf(t, !metadata.Migrations["fyneDo"] || !metadata.Migrations["other"], "migration metadata = %#v", metadata.Migrations)
}

func TestFyneDoMigrationInitializesMissingMigrationMetadata(t *testing.T) {
	t.Parallel()
	metadata := withFyneDoMigration(fyne.AppMetadata{})
	testutil.ErrorIf(t, !metadata.Migrations["fyneDo"], "migration metadata = %#v", metadata.Migrations)
}

func TestStartupConfigSelectsEverySupportedActor(t *testing.T) {
	t.Parallel()
	for _, actor := range []string{"owner", "manager", "sommelier", "bartender", "anonymous"} {
		t.Run(actor, func(t *testing.T) {
			t.Parallel()
			config, err := startupConfig([]string{"-actor", actor}, new(bytes.Buffer))
			testutil.ErrorIf(t, err != nil || config == nil || config.actor != actor || config.dataDirectory == "" || config.databasePath != defaultDatabasePath, "startup config = %#v, %v", config, err)
		})
	}
}

func TestStartupConfigDefaultsFreshDesktopToOwnerAndSupportsAlias(t *testing.T) {
	t.Parallel()
	config, err := startupConfig(nil, new(bytes.Buffer))
	testutil.ErrorIf(t, err != nil || config == nil || config.actor != "owner" || config.databasePath != defaultDatabasePath, "default startup config = %#v, %v", config, err)
	config, err = startupConfig([]string{"-as", "bartender"}, new(bytes.Buffer))
	testutil.ErrorIf(t, err != nil || config == nil || config.actor != "bartender", "alias startup config = %#v, %v", config, err)
}

func TestStartupConfigHelpAndInvalidActor(t *testing.T) {
	t.Parallel()
	output := new(bytes.Buffer)
	config, err := startupConfig([]string{"-help"}, output)
	testutil.ErrorIf(t, err != nil || config != nil || !strings.Contains(output.String(), "owner|manager|sommelier|bartender|anonymous"), "help = %#v, %v, %q", config, err, output.String())
	{
		_, err := startupConfig([]string{"-actor", "intruder"}, new(bytes.Buffer))
		testutil.ErrorIf(t, err == nil, "%v", "invalid actor accepted")
	}
}
