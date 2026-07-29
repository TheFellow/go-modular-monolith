package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestStartupConfigSelectsEverySupportedActor(t *testing.T) {
	for _, actor := range []string{"owner", "manager", "sommelier", "bartender", "anonymous"} {
		t.Run(actor, func(t *testing.T) {
			config, err := startupConfig([]string{"-actor", actor}, new(bytes.Buffer))
			if err != nil || config == nil || config.actor != actor || config.dataDirectory == "" {
				t.Fatalf("startup config = %#v, %v", config, err)
			}
		})
	}
}

func TestStartupConfigDefaultsFreshDesktopToOwnerAndSupportsAlias(t *testing.T) {
	config, err := startupConfig(nil, new(bytes.Buffer))
	if err != nil || config == nil || config.actor != "owner" {
		t.Fatalf("default startup config = %#v, %v", config, err)
	}
	config, err = startupConfig([]string{"-as", "bartender"}, new(bytes.Buffer))
	if err != nil || config == nil || config.actor != "bartender" {
		t.Fatalf("alias startup config = %#v, %v", config, err)
	}
}

func TestStartupConfigHelpAndInvalidActor(t *testing.T) {
	output := new(bytes.Buffer)
	config, err := startupConfig([]string{"-help"}, output)
	if err != nil || config != nil || !strings.Contains(output.String(), "owner|manager|sommelier|bartender|anonymous") {
		t.Fatalf("help = %#v, %v, %q", config, err, output.String())
	}
	if _, err := startupConfig([]string{"-actor", "intruder"}, new(bytes.Buffer)); err == nil {
		t.Fatal("invalid actor accepted")
	}
}
