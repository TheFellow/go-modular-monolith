package main

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/TheFellow/go-modular-monolith/app/domains/ingredients/models"
	"github.com/TheFellow/go-modular-monolith/pkg/filter"
)

func TestFilterHelpUsesConcreteSchema(t *testing.T) {
	var out bytes.Buffer
	if err := writeFilterHelp(&out, models.ListFilterSchema()); err != nil {
		t.Fatal(err)
	}
	text := out.String()
	for _, want := range []string{
		"category", "Ingredient category", "&& / and", "value.contains", `--filter 'category == "spirit"`,
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("help does not contain %q:\n%s", want, text)
		}
	}
	for _, example := range models.ListFilterSchema().Examples() {
		if _, err := filter.Parse(models.ListFilterSchema(), example); err != nil {
			t.Fatalf("generated example %q does not parse: %v", example, err)
		}
	}
}

func TestFilterHelpDoesNotOpenApplication(t *testing.T) {
	c, err := NewCLI()
	if err != nil {
		t.Fatal(err)
	}
	c.dbPath = t.TempDir() // opening a directory would fail if Before reached storage
	cmd := c.Command()
	var out bytes.Buffer
	cmd.Writer = &out
	cmd.ErrWriter = &out
	cmd.Command("ingredients").Command("list").Writer = &out
	if err := cmd.Run(context.Background(), []string{"mixology", "ingredients", "list", "--filter-help"}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "FILTER SYNTAX") {
		t.Fatalf("unexpected output:\n%s", out.String())
	}
}

func TestEveryListCommandHasFilterFlags(t *testing.T) {
	c, err := NewCLI()
	if err != nil {
		t.Fatal(err)
	}
	for _, noun := range c.Command().Commands {
		for _, command := range noun.Commands {
			if command.Name != "list" {
				continue
			}
			names := map[string]bool{}
			for _, flag := range command.Flags {
				names[flag.Names()[0]] = true
			}
			if !names["filter"] || !names["filter-help"] {
				t.Errorf("%s list filter flags = %v", noun.Name, names)
			}
		}
	}
}
